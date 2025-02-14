package bunexp

import (
	"context"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun"
)

type DBReplica interface {
	bun.IConn
	PingContext(context.Context) error
	Close() error
}

type DBReplicaRole int

const (
	DBReplicaReadOnly DBReplicaRole = 1 << iota
	DBReplicaBackup
)

// TODO:
//   - make monitoring interval configurable
//   - make ping timeout configutable
//   - allow adding read/write replicas for multi-master replication
type ReadWriteConnResolver struct {
	rw     replicas // for read-write queries
	ro     replicas // for read-only queries
	closed atomic.Bool
}

func NewReadWriteConnResolver(opts ...ReadWriteConnResolverOption) *ReadWriteConnResolver {
	r := new(ReadWriteConnResolver)

	for _, opt := range opts {
		opt(r)
	}

	r.rw.startMonitoring(&r.closed)
	r.ro.startMonitoring(&r.closed)

	return r
}

type ReadWriteConnResolverOption func(r *ReadWriteConnResolver)

func WithDBReplica(db DBReplica, roles ...DBReplicaRole) ReadWriteConnResolverOption {
	var role DBReplicaRole
	for _, r := range roles {
		role |= r
	}

	return func(r *ReadWriteConnResolver) {
		if role&DBReplicaReadOnly == 0 {
			r.rw.add(db, role)
		}
		r.ro.add(db, role)
	}
}

func (r *ReadWriteConnResolver) Close() error {
	if r.closed.Swap(true) {
		return nil
	}
	var firstErr error
	if err := r.rw.close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := r.ro.close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (r *ReadWriteConnResolver) ResolveConn(query bun.Query) bun.IConn {
	readOnly := bun.IsReadOnlyQuery(query)

	var replicas []replica
	if readOnly {
		replicas = r.ro.healthyReplicas()
	} else {
		replicas = r.rw.healthyReplicas()
	}

	switch len(replicas) {
	case 0:
		return nil
	case 1:
		return replicas[0].DBReplica
	}

	var i int
	if readOnly {
		i = int(r.ro.next.Add(1))
	} else {
		i = int(r.rw.next.Add(1))
	}
	return replicas[i%len(replicas)]
}

type replicas struct {
	replicas []replica // read-only replicas
	healthy  atomic.Pointer[[]replica]
	next     atomic.Int64
}

type replica struct {
	DBReplica
	roles DBReplicaRole
}

func (r *replicas) add(db DBReplica, roles DBReplicaRole) {
	r.replicas = append(r.replicas, replica{
		DBReplica: db,
		roles:     roles,
	})
}

func (r *replicas) healthyReplicas() []replica {
	if ptr := r.healthy.Load(); ptr != nil {
		return *ptr
	}
	return nil
}

func (r *replicas) close() error {
	var firstErr error
	for _, db := range r.replicas {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *replicas) startMonitoring(closed *atomic.Bool) {
	if len(r.replicas) == 0 {
		return
	}

	r.healthy.Store(&r.replicas)
	go r.monitor(closed)

	// Start with a random replica.
	rnd := rand.IntN(len(r.replicas))
	r.next.Store(int64(rnd))
}

func (r *replicas) monitor(closed *atomic.Bool) {
	const interval = 5 * time.Second
	for !closed.Load() {
		healthy := make([]replica, 0, len(r.replicas))

		for _, replica := range r.replicas {
			if replica.roles&DBReplicaBackup != 0 {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := replica.PingContext(ctx)
			cancel()

			if err == nil {
				healthy = append(healthy, replica)
			}
		}

		if len(healthy) == 0 {
			healthy = r.replicas
		}

		r.healthy.Store(&healthy)
		time.Sleep(interval)
	}
}
