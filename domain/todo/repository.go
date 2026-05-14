package todo

import "context"

// レポジトリはtodoの永続化を抽象化するインターフェース

// ドメイン層にインターフェースを置き、実装はインフラにおく
// ドメイン層をdbなどの技術的詳細から独立させる

// goでは使う側でインターフェースを定義するのが普通なので、この配置も理にかなってる

type Repository interface {
	// 既存ならupdate , 新規ならinsert をリポジトリ実装が判断する
	// 呼び出し側はそれを知らなくて良い
	Save(ctx context.Context, t *Todo) error

	// FindByIDはIDで一件取得する
	FindByID(ctx context.Context, id ID) (*Todo, error)

	// DeleteでID削除
	Delete(ctx context.Context, id ID) error
}
