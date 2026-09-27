package update

import "context"

type Git interface {
	CurrentBranch(context.Context, string) (string, error)
	Head(context.Context, string) (string, error)
	Pull(context.Context, string) error
}
