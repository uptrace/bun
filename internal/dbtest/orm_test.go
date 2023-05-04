package dbtest_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dbfixture"
	"github.com/TommyLeng/bun/dialect/feature"
)

func TestORM(t *testing.T) {
	type Test struct {
		fn func(*testing.T, *bun.DB)
	}

	tests := []Test{
		{testBookRelations},
		{testAuthorRelations},
		{testGenreRelations},
		{testTranslationRelations},
		{testBulkUpdate},
		{testRelationColumn},
		{testRelationExcludeAll},
		{testM2MRelationExcludeColumn},
		{testRelationBelongsToSelf},
		{testCompositeHasMany},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		createTestSchema(t, db)

		for _, test := range tests {
			loadTestData(t, ctx, db)

			t.Run(funcName(test.fn), func(t *testing.T) {
				test.fn(t, db)
			})
		}
	})
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
		OrderExpr("author.id ASC").
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

func testBulkUpdate(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

	var books []Book
	err := db.NewSelect().Model(&books).Scan(ctx)
	require.NoError(t, err)

	res, err := db.NewUpdate().
		With("_data", db.NewValues(&books)).
		Model((*Book)(nil)).
		Table("_data").
		Apply(func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				SetColumn("title", "UPPER(?)", q.FQN("title")).
				Where("? = _data.id", q.FQN("id"))
		}).
		Exec(ctx)
	require.NoError(t, err)

	n, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, len(books), int(n))

	var books2 []Book
	err = db.NewSelect().Model(&books2).Scan(ctx)
	require.NoError(t, err)

	for i := range books {
		require.Equal(t, strings.ToUpper(books[i].Title), books2[i].Title)
	}
}

func testRelationColumn(t *testing.T, db *bun.DB) {
	book := new(Book)
	err := db.NewSelect().
		Model(book).
		ExcludeColumn("created_at").
		Relation("Author", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Column("name")
		}).
		OrderExpr("book.id").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, &Book{
		ID:       100,
		Title:    "book 1",
		AuthorID: 10,
		Author: Author{
			Name: "author 1",
		},
		EditorID: 11,
	}, book)
}

func testRelationExcludeAll(t *testing.T, db *bun.DB) {
	book := new(Book)
	err := db.NewSelect().
		Model(book).
		ExcludeColumn("created_at").
		Relation("Author", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ExcludeColumn("*")
		}).
		Relation("Author.Avatar").
		Relation("Editor").
		OrderExpr("book.id").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, &Book{
		ID:       100,
		Title:    "book 1",
		AuthorID: 10,
		Author: Author{
			Avatar: Image{
				ID:   1,
				Path: "/path/to/1.jpg",
			},
		},
		EditorID: 11,
		Editor: &Author{
			ID:       11,
			Name:     "author 2",
			AvatarID: 2,
		},
	}, book)

	book = new(Book)
	err = db.NewSelect().
		Model(book).
		ExcludeColumn("*").
		Relation("Author", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.ExcludeColumn("*")
		}).
		Relation("Author.Avatar").
		Relation("Editor").
		OrderExpr("book.id").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, &Book{
		Author: Author{
			Avatar: Image{
				ID:   1,
				Path: "/path/to/1.jpg",
			},
		},
		Editor: &Author{
			ID:       11,
			Name:     "author 2",
			AvatarID: 2,
		},
	}, book)
}

func testRelationBelongsToSelf(t *testing.T, db *bun.DB) {
	type Model struct {
		bun.BaseModel `bun:"alias:m"`

		ID      int64 `bun:",pk,autoincrement"`
		ModelID int64
		Model   *Model `bun:"rel:belongs-to"`
	}

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	models := []Model{
		{ID: 1},
		{ID: 2, ModelID: 1},
	}
	_, err = db.NewInsert().Model(&models).Exec(ctx)
	require.NoError(t, err)

	models = nil
	err = db.NewSelect().Model(&models).Relation("Model").OrderExpr("m.id ASC").Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, []Model{
		{ID: 1},
		{ID: 2, ModelID: 1, Model: &Model{ID: 1}},
	}, models)
}

func testM2MRelationExcludeColumn(t *testing.T, db *bun.DB) {
	type Item struct {
		ID        int64     `bun:",pk,autoincrement"`
		CreatedAt time.Time `bun:",notnull,nullzero"`
		UpdatedAt time.Time `bun:",notnull,nullzero"`
	}

	type Order struct {
		ID    int64 `bun:",pk,autoincrement"`
		Text  string
		Items []Item `bun:"m2m:order_to_items"`
	}

	type OrderToItem struct {
		OrderID   int64     `bun:",pk"`
		Order     *Order    `bun:"rel:has-one,join:order_id=id"`
		ItemID    int64     `bun:",pk"`
		Item      *Item     `bun:"rel:has-one,join:item_id=id"`
		CreatedAt time.Time `bun:",notnull,nullzero"`
	}

	db.RegisterModel((*OrderToItem)(nil))

	err := db.ResetModel(ctx, (*Order)(nil), (*Item)(nil), (*OrderToItem)(nil))
	require.NoError(t, err)

	items := []Item{
		{ID: 1, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)},
		{ID: 2, CreatedAt: time.Unix(2, 0), UpdatedAt: time.Unix(1, 0)},
	}
	_, err = db.NewInsert().Model(&items).Exec(ctx)
	require.NoError(t, err)

	orders := []Order{
		{ID: 1},
		{ID: 2},
	}
	_, err = db.NewInsert().Model(&orders).Exec(ctx)
	require.NoError(t, err)

	orderItems := []OrderToItem{
		{OrderID: 1, ItemID: 1, CreatedAt: time.Unix(3, 0)},
		{OrderID: 2, ItemID: 2, CreatedAt: time.Unix(4, 0)},
	}
	_, err = db.NewInsert().Model(&orderItems).Exec(ctx)
	require.NoError(t, err)

	order := new(Order)
	err = db.NewSelect().
		Model(order).
		Where("id = ?", 1).
		Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.ExcludeColumn("updated_at")
		}).
		Scan(ctx)
	require.NoError(t, err)
}

func testCompositeHasMany(t *testing.T, db *bun.DB) {
	department := new(Department)
	err := db.NewSelect().
		Model(department).
		Where("company_no=? AND no=?", "company one", "hr").
		Relation("Employees").
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, "hr", department.No)
	require.Equal(t, 2, len(department.Employees))
}

type Genre struct {
	ID     int `bun:",pk"`
	Name   string
	Rating int `bun:",scanonly"`

	Books []Book `bun:"m2m:book_genres"`

	ParentID  int
	Subgenres []Genre `bun:"rel:has-many,join:id=parent_id"`
}

func (g Genre) String() string {
	return fmt.Sprintf("Genre<Id=%d Name=%q>", g.ID, g.Name)
}

type Image struct {
	ID   int `bun:",pk"`
	Path string
}

type Author struct {
	ID    int     `bun:",pk"`
	Name  string  `bun:",unique"`
	Books []*Book `bun:"rel:has-many"`

	AvatarID int
	Avatar   Image `bun:"rel:belongs-to"`
}

func (a Author) String() string {
	return fmt.Sprintf("Author<ID=%d Name=%q>", a.ID, a.Name)
}

var _ bun.BeforeAppendModelHook = (*Author)(nil)

func (*Author) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return nil
}

type BookGenre struct {
	bun.BaseModel `bun:"alias:bg"` // custom table alias

	BookID  int    `bun:",pk"`
	Book    *Book  `bun:"rel:belongs-to"`
	GenreID int    `bun:",pk"`
	Genre   *Genre `bun:"rel:belongs-to"`

	Genre_Rating int // is copied to Genre.Rating
}

type Book struct {
	ID        int `bun:",pk"`
	Title     string
	AuthorID  int
	Author    Author `bun:"rel:belongs-to"`
	EditorID  int
	Editor    *Author   `bun:"rel:belongs-to"`
	CreatedAt time.Time `bun:",nullzero,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero"`

	Genres       []Genre       `bun:"m2m:book_genres"` // many to many relation
	Translations []Translation `bun:"rel:has-many"`
	Comments     []Comment     `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
}

func (b Book) String() string {
	return fmt.Sprintf("Book<Id=%d Title=%q>", b.ID, b.Title)
}

var _ bun.BeforeAppendModelHook = (*Book)(nil)

func (*Book) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	return nil
}

// BookWithCommentCount is like Book model, but has additional CommentCount
// field that is used to select data into it. The use of `bun:",extend"` tag
// is essential here.
type BookWithCommentCount struct {
	Book `bun:",extend"`

	CommentCount int
}

type Translation struct {
	bun.BaseModel `bun:"alias:tr"`

	ID     int    `bun:",pk"`
	BookID int    `bun:"unique:book_id_lang"`
	Book   *Book  `bun:"rel:belongs-to"`
	Lang   string `bun:"unique:book_id_lang"`

	Comments []Comment `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
}

type Comment struct {
	TrackableID   int    // Book.ID or Translation.ID
	TrackableType string // "book" or "translation"
	Text          string
}

type Department struct {
	bun.BaseModel `bun:"alias:d"`
	CompanyNo     string     `bun:",pk"`
	No            string     `bun:",pk"`
	Employees     []Employee `bun:"rel:has-many,join:company_no=company_no,join:no=department_no"`
}

type Employee struct {
	bun.BaseModel `bun:"alias:p"`
	CompanyNo     string `bun:",pk"`
	DepartmentNo  string `bun:",pk"`
	Name          string `bun:",pk"`
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
		(*Department)(nil),
		(*Employee)(nil),
	}
	for _, model := range models {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)

		_, err = db.NewCreateTable().Model(model).Exec(ctx)
		require.NoError(t, err)
	}
}

func loadTestData(t *testing.T, ctx context.Context, db *bun.DB) {
	fixture := dbfixture.New(db, dbfixture.WithTruncateTables())
	err := fixture.Load(ctx, os.DirFS("testdata"), "fixture.yaml")
	require.NoError(t, err)
}
