package pgdriver

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun"
)

const pingChannel = "bun:ping"

var (
	errListenerClosed = errors.New("bun: listener is closed")
	errPingTimeout    = errors.New("bun: ping timeout")
)

type Listener struct {
	db     *bun.DB
	driver *driverConnector

	channels []string

	mu     sync.Mutex
	cn     *Conn
	closed bool
	exit   chan struct{}
}

func NewListener(db *bun.DB) *Listener {
	return &Listener{
		db:     db,
		driver: db.Driver().(Driver).connector,
		exit:   make(chan struct{}),
	}
}

// Close closes the listener, releasing any open resources.
func (ln *Listener) Close() error {
	return ln.withLock(func() error {
		if ln.closed {
			return errListenerClosed
		}

		ln.closed = true
		close(ln.exit)

		return ln.closeConn(errListenerClosed)
	})
}

func (ln *Listener) withLock(fn func() error) error {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	return fn()
}

func (ln *Listener) conn(ctx context.Context) (*Conn, error) {
	if ln.closed {
		return nil, errListenerClosed
	}
	if ln.cn != nil {
		return ln.cn, nil
	}

	atomic.AddUint64(&ln.driver.stats.Queries, 1)

	cn, err := ln._conn(ctx)
	if err != nil {
		atomic.AddUint64(&ln.driver.stats.Errors, 1)
		return nil, err
	}

	ln.cn = cn
	return cn, nil
}

func (ln *Listener) _conn(ctx context.Context) (*Conn, error) {
	driverConn, err := ln.driver.Connect(ctx)
	if err != nil {
		return nil, err
	}
	cn := driverConn.(*Conn)

	if len(ln.channels) > 0 {
		err := ln.listen(ctx, cn, ln.channels...)
		if err != nil {
			_ = cn.Close()
			return nil, err
		}
	}

	return cn, nil
}

func (ln *Listener) checkConn(ctx context.Context, cn *Conn, err error, allowTimeout bool) {
	_ = ln.withLock(func() error {
		if ln.closed || ln.cn != cn {
			return nil
		}
		if isBadConn(err, allowTimeout) {
			ln.reconnect(ctx, err)
		}
		return nil
	})
}

func (ln *Listener) reconnect(ctx context.Context, reason error) {
	if ln.cn != nil {
		Logger.Printf(ctx, "bun: discarding bad listener connection: %s", reason)
		_ = ln.closeConn(reason)
	}
	_, _ = ln.conn(ctx)
}

func (ln *Listener) closeConn(reason error) error {
	if ln.cn == nil {
		return nil
	}
	err := ln.cn.Close()
	ln.cn = nil
	return err
}

// Listen starts listening for notifications on channels.
func (ln *Listener) Listen(ctx context.Context, channels ...string) error {
	var cn *Conn

	if err := ln.withLock(func() error {
		ln.channels = appendIfNotExists(ln.channels, channels...)

		var err error
		cn, err = ln.conn(ctx)
		return err
	}); err != nil {
		return err
	}

	if err := ln.listen(ctx, cn, channels...); err != nil {
		ln.checkConn(ctx, cn, err, false)
		return err
	}
	return nil
}

func (ln *Listener) listen(ctx context.Context, cn *Conn, channels ...string) error {
	for _, channel := range channels {
		if err := writeQuery(ctx, cn, "LISTEN "+strconv.Quote(channel)); err != nil {
			return err
		}
	}
	return nil
}

// Unlisten stops listening for notifications on channels.
func (ln *Listener) Unlisten(ctx context.Context, channels ...string) error {
	var cn *Conn

	if err := ln.withLock(func() error {
		ln.channels = removeIfExists(ln.channels, channels...)

		var err error
		cn, err = ln.conn(ctx)
		return err
	}); err != nil {
		return err
	}

	if err := ln.unlisten(ctx, cn, channels...); err != nil {
		ln.checkConn(ctx, cn, err, false)
		return err
	}
	return nil
}

func (ln *Listener) unlisten(ctx context.Context, cn *Conn, channels ...string) error {
	for _, channel := range channels {
		if err := writeQuery(ctx, cn, "UNLISTEN "+strconv.Quote(channel)); err != nil {
			return err
		}
	}
	return nil
}

// Receive indefinitely waits for a notification. This is low-level API
// and in most cases Channel should be used instead.
func (ln *Listener) Receive(ctx context.Context) (channel string, payload string, err error) {
	return ln.ReceiveTimeout(ctx, 0)
}

// ReceiveTimeout waits for a notification until timeout is reached.
// This is low-level API and in most cases Channel should be used instead.
func (ln *Listener) ReceiveTimeout(
	ctx context.Context, timeout time.Duration,
) (channel, payload string, err error) {
	var cn *Conn

	if err := ln.withLock(func() error {
		var err error
		cn, err = ln.conn(ctx)
		return err
	}); err != nil {
		return "", "", err
	}

	cn.setReadDeadline(ctx, timeout)
	channel, payload, err = readNotification(cn)
	if err != nil {
		ln.checkConn(ctx, cn, err, timeout > 0)
		return "", "", err
	}

	return channel, payload, nil
}

// Channel returns a channel for concurrently receiving notifications.
// It periodically sends Ping notification to test connection health.
//
// The channel is closed with Listener. Receive* APIs can not be used
// after channel is created.
func (ln *Listener) Channel(opts ...ChannelOption) *Channel {
	return newChannel(ln, opts)
}

//------------------------------------------------------------------------------

// Notification received with LISTEN command.
type Notification struct {
	Channel string
	Payload string
}

type ChannelOption func(c *Channel)

func WithChannelSize(size int) ChannelOption {
	return func(c *Channel) {
		c.size = size
	}
}

type Channel struct {
	ctx context.Context
	ln  *Listener

	size            int
	pingTimeout     time.Duration
	chanSendTimeout time.Duration

	C      chan Notification
	pingCh chan struct{}
}

func newChannel(ln *Listener, opts []ChannelOption) *Channel {
	c := &Channel{
		ctx: context.TODO(),
		ln:  ln,

		size:            100,
		pingTimeout:     time.Second,
		chanSendTimeout: time.Minute,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.C = make(chan Notification, c.size)
	c.pingCh = make(chan struct{}, 1)
	_ = c.ln.Listen(c.ctx, pingChannel)
	go c.startReceive()
	go c.startPing()

	return c
}

func (c *Channel) startReceive() {
	timer := time.NewTimer(time.Minute)
	timer.Stop()

	var errCount int
	for {
		channel, payload, err := c.ln.Receive(c.ctx)
		if err != nil {
			if err == errListenerClosed {
				close(c.C)
				return
			}

			if errCount > 0 {
				time.Sleep(500 * time.Millisecond)
			}
			errCount++

			continue
		}

		errCount = 0

		// Any notification is as good as a ping.
		select {
		case c.pingCh <- struct{}{}:
		default:
		}

		switch channel {
		case pingChannel:
			// ignore
		default:
			timer.Reset(c.chanSendTimeout)
			select {
			case c.C <- Notification{channel, payload}:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				Logger.Printf(
					c.ctx,
					"pgdriver: %s channel is full for %s (notification is dropped)",
					c,
					c.chanSendTimeout,
				)
			}
		}
	}
}

func (c *Channel) startPing() {
	timer := time.NewTimer(time.Minute)
	timer.Stop()

	healthy := true
	for {
		timer.Reset(c.pingTimeout)
		select {
		case <-c.pingCh:
			healthy = true
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			pingErr := c.ping(c.ctx)
			if healthy {
				healthy = false
			} else {
				if pingErr == nil {
					pingErr = errPingTimeout
				}
				_ = c.ln.withLock(func() error {
					c.ln.reconnect(c.ctx, pingErr)
					return nil
				})
			}
		case <-c.ln.exit:
			return
		}
	}
}

func (c *Channel) ping(ctx context.Context) error {
	_, err := c.ln.db.ExecContext(ctx, "NOTIFY "+strconv.Quote(pingChannel))
	return err
}

func appendIfNotExists(ss []string, es ...string) []string {
loop:
	for _, e := range es {
		for _, s := range ss {
			if s == e {
				continue loop
			}
		}
		ss = append(ss, e)
	}
	return ss
}

func removeIfExists(ss []string, es ...string) []string {
	for _, e := range es {
		for i, s := range ss {
			if s == e {
				last := len(ss) - 1
				ss[i] = ss[last]
				ss = ss[:last]
				break
			}
		}
	}
	return ss
}
