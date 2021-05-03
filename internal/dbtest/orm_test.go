package dbtest_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestORM(t *testing.T) {
	type Test struct {
		name string
		fn   func(*testing.T, *bun.DB)
	}

	tests := []Test{
		{"testBookRelations", testBookRelations},
		{"testAuthorRelations", testAuthorRelations},
		{"testGenreRelations", testGenreRelations},
		{"testTranslationRelations", testTranslationRelations},
	}

	for _, db := range dbs(t) {
		t.Run(db.Dialect().Name(), func(t *testing.T) {
			// defer db.Close()

			createTestSchema(t, db)
			loadTestData(t, db)

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					test.fn(t, db)
				})
			}
		})
	}
}

func testBookRelations(t *testing.T, db *bun.DB) {
	book := new(Book)
	err := db.NewSelect().
		Model(book).
		Column("book.id").
		Relation("Author").
		Relation("Author.Avatar").
		Relation("Editor").
		Relation("Editor.Avatar").
		Relation("Genres").
		Relation("Comments").
		Relation("Translations", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("id")
		}).
		Relation("Translations.Comments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("text")
		}).
		OrderExpr("book.id ASC").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, 100, book.ID)
	require.Equal(t, &Book{
		ID:    100,
		Title: "",
		Author: Author{
			ID:       10,
			Name:     "author 1",
			AvatarID: 1,
			Avatar: Image{
				ID:   1,
				Path: "/path/to/1.jpg",
			},
		},
		Editor: &Author{
			ID:       11,
			Name:     "author 2",
			AvatarID: 2,
			Avatar: Image{
				ID:   2,
				Path: "/path/to/2.jpg",
			},
		},
		CreatedAt: time.Time{},
		Genres: []Genre{
			{ID: 1, Name: "genre 1", Rating: 999},
			{ID: 2, Name: "genre 2", Rating: 9999},
		},
		Translations: []Translation{{
			ID:     1000,
			BookID: 100,
			Lang:   "ru",
			Comments: []Comment{
				{TrackableID: 1000, TrackableType: "translation", Text: "comment3"},
			},
		}, {
			ID:       1001,
			BookID:   100,
			Lang:     "md",
			Comments: nil,
		}},
		Comments: []Comment{
			{TrackableID: 100, TrackableType: "book", Text: "comment1"},
			{TrackableID: 100, TrackableType: "book", Text: "comment2"},
		},
	}, book)
}

func testAuthorRelations(t *testing.T, db *bun.DB) {
	var author Author
	err := db.NewSelect().
		Model(&author).
		Column("author.*").
		Relation("Books", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Column("book.id", "book.author_id", "book.editor_id").OrderExpr("book.id ASC")
		}).
		Relation("Books.Author").
		Relation("Books.Editor").
		Relation("Books.Translations", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("tr.id ASC")
		}).
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, Author{
		ID:       10,
		Name:     "author 1",
		AvatarID: 1,
		Books: []*Book{{
			ID:        100,
			Title:     "",
			AuthorID:  10,
			Author:    Author{ID: 10, Name: "author 1", AvatarID: 1},
			EditorID:  11,
			Editor:    &Author{ID: 11, Name: "author 2", AvatarID: 2},
			CreatedAt: time.Time{},
			Genres:    nil,
			Translations: []Translation{
				{ID: 1000, BookID: 100, Book: nil, Lang: "ru", Comments: nil},
				{ID: 1001, BookID: 100, Book: nil, Lang: "md", Comments: nil},
			},
		}, {
			ID:        101,
			Title:     "",
			AuthorID:  10,
			Author:    Author{ID: 10, Name: "author 1", AvatarID: 1},
			EditorID:  12,
			Editor:    &Author{ID: 12, Name: "author 3", AvatarID: 3},
			CreatedAt: time.Time{},
			Genres:    nil,
			Translations: []Translation{
				{ID: 1002, BookID: 101, Book: nil, Lang: "ua", Comments: nil},
			},
		}},
	}, author)
}

func testGenreRelations(t *testing.T, db *bun.DB) {
	var genre Genre
	err := db.NewSelect().
		Model(&genre).
		Column("genre.*").
		Relation("Books", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ColumnExpr("book.id")
		}).
		Relation("Books.Translations", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("tr.id ASC")
		}).
		OrderExpr("genre.id ASC").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, Genre{
		ID:     1,
		Name:   "genre 1",
		Rating: 0,
		Books: []Book{{
			ID: 100,
			Translations: []Translation{
				{ID: 1000, BookID: 100, Book: nil, Lang: "ru", Comments: nil},
				{ID: 1001, BookID: 100, Book: nil, Lang: "md", Comments: nil},
			},
		}, {
			ID: 101,
			Translations: []Translation{
				{ID: 1002, BookID: 101, Book: nil, Lang: "ua", Comments: nil},
			},
		}},
		ParentID:  0,
		Subgenres: nil,
	}, genre)
}

func testTranslationRelations(t *testing.T, db *bun.DB) {
	var translation Translation
	err := db.NewSelect().
		Model(&translation).
		Column("tr.*").
		Relation("Book", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ColumnExpr("book.id AS book__id")
		}).
		Relation("Book.Author").
		Relation("Book.Editor").
		OrderExpr("tr.id ASC").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, Translation{
		ID:     1000,
		BookID: 100,
		Book: &Book{
			ID:     100,
			Author: Author{ID: 10, Name: "author 1", AvatarID: 1},
			Editor: &Author{ID: 11, Name: "author 2", AvatarID: 2},
		},
		Lang: "ru",
	}, translation)
}

type Genre struct {
	ID     int
	Name   string
	Rating int `bun:"-"`

	Books []Book `bun:"m2m:book_genres"`

	ParentID  int
	Subgenres []Genre `bun:"rel:has-many,join:id=parent_id"`
}

func (g Genre) String() string {
	return fmt.Sprintf("Genre<Id=%d Name=%q>", g.ID, g.Name)
}

type Image struct {
	ID   int
	Path string
}

type Author struct {
	ID    int
	Name  string  `bun:",unique"`
	Books []*Book `bun:"rel:has-many"`

	AvatarID int
	Avatar   Image `bun:"rel:has-one"`
}

func (a Author) String() string {
	return fmt.Sprintf("Author<ID=%d Name=%q>", a.ID, a.Name)
}

type BookGenre struct {
	bun.BaseTable `bun:"alias:bg"` // custom table alias

	BookID  int    `bun:",pk"`
	Book    *Book  `bun:"rel:has-one"`
	GenreID int    `bun:",pk"`
	Genre   *Genre `bun:"rel:has-one"`

	Genre_Rating int // is copied to Genre.Rating
}

type Book struct {
	ID        int
	Title     string
	AuthorID  int
	Author    Author `bun:"rel:has-one"`
	EditorID  int
	Editor    *Author   `bun:"rel:has-one"`
	CreatedAt time.Time `bun:"default:current_timestamp"`
	UpdatedAt time.Time

	Genres       []Genre       `bun:"m2m:book_genres"` // many to many relation
	Translations []Translation `bun:"rel:has-many"`
	Comments     []Comment     `bun:"rel:has-many,join:'id=trackable_id,type=trackable_type',polymorphic"`
}

func (b Book) String() string {
	return fmt.Sprintf("Book<Id=%d Title=%q>", b.ID, b.Title)
}

// BookWithCommentCount is like Book model, but has additional CommentCount
// field that is used to select data into it. The use of `bun:",inherit"` tag
// is essential here so it inherits internal model properties such as table name.
type BookWithCommentCount struct {
	Book `bun:",inherit"`

	CommentCount int
}

type Translation struct {
	bun.BaseTable `bun:"alias:tr"`

	ID     int
	BookID int    `bun:"unique:book_id_lang"`
	Book   *Book  `bun:"rel:has-one"`
	Lang   string `bun:"unique:book_id_lang"`

	Comments []Comment `bun:"rel:has-many,join:'id=trackable_id,type=trackable_type',polymorphic"`
}

type Comment struct {
	TrackableID   int    // Book.ID or Translation.ID
	TrackableType string // "Book" or "Translation"
	Text          string
}

func createTestSchema(t *testing.T, db *bun.DB) {
	_ = db.Table(reflect.TypeOf((*BookGenre)(nil)).Elem())

	models := []interface{}{
		(*Image)(nil),
		(*Author)(nil),
		(*Book)(nil),
		(*Genre)(nil),
		(*BookGenre)(nil),
		(*Translation)(nil),
		(*Comment)(nil),
	}
	for _, model := range models {
		_, err := db.NewDropTable().Model(model).IfExists().Cascade().Exec(ctx)
		require.NoError(t, err)

		_, err = db.NewCreateTable().Model(model).Exec(ctx)
		require.NoError(t, err)
	}
}

func loadTestData(t *testing.T, db *bun.DB) {
	genres := []Genre{{
		ID:   1,
		Name: "genre 1",
	}, {
		ID:   2,
		Name: "genre 2",
	}, {
		ID:       3,
		Name:     "subgenre 1",
		ParentID: 1,
	}, {
		ID:       4,
		Name:     "subgenre 2",
		ParentID: 1,
	}}
	_, err := db.NewInsert().Model(&genres).Exec(ctx)
	require.NoError(t, err)

	images := []Image{{
		ID:   1,
		Path: "/path/to/1.jpg",
	}, {
		ID:   2,
		Path: "/path/to/2.jpg",
	}, {
		ID:   3,
		Path: "/path/to/3.jpg",
	}}
	_, err = db.NewInsert().Model(&images).Exec(ctx)
	require.NoError(t, err)

	authors := []Author{{
		ID:       10,
		Name:     "author 1",
		AvatarID: images[0].ID,
	}, {
		ID:       11,
		Name:     "author 2",
		AvatarID: images[1].ID,
	}, {
		ID:       12,
		Name:     "author 3",
		AvatarID: images[2].ID,
	}}
	_, err = db.NewInsert().Model(&authors).Exec(ctx)
	require.NoError(t, err)

	books := []Book{{
		ID:       100,
		Title:    "book 1",
		AuthorID: 10,
		EditorID: 11,
	}, {
		ID:       101,
		Title:    "book 2",
		AuthorID: 10,
		EditorID: 12,
	}, {
		ID:       102,
		Title:    "book 3",
		AuthorID: 11,
		EditorID: 11,
	}}
	_, err = db.NewInsert().Model(&books).Exec(ctx)
	require.NoError(t, err)

	bookGenres := []BookGenre{{
		BookID:       100,
		GenreID:      1,
		Genre_Rating: 999,
	}, {
		BookID:       100,
		GenreID:      2,
		Genre_Rating: 9999,
	}, {
		BookID:       101,
		GenreID:      1,
		Genre_Rating: 99999,
	}}
	_, err = db.NewInsert().Model(&bookGenres).Exec(ctx)
	require.NoError(t, err)

	translations := []Translation{{
		ID:     1000,
		BookID: 100,
		Lang:   "ru",
	}, {
		ID:     1001,
		BookID: 100,
		Lang:   "md",
	}, {
		ID:     1002,
		BookID: 101,
		Lang:   "ua",
	}}
	_, err = db.NewInsert().Model(&translations).Exec(ctx)
	require.NoError(t, err)

	comments := []Comment{{
		TrackableID:   100,
		TrackableType: "book",
		Text:          "comment1",
	}, {
		TrackableID:   100,
		TrackableType: "book",
		Text:          "comment2",
	}, {
		TrackableID:   1000,
		TrackableType: "translation",
		Text:          "comment3",
	}}
	_, err = db.NewInsert().Model(&comments).Exec(ctx)
	require.NoError(t, err)
}
