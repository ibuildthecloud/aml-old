package eval

import "context"

type tickCounter struct{}

func WithTicks(ctx context.Context, ticks int) context.Context {
	return context.WithValue(ctx, tickCounter{}, &ticks)
}

func tick(ctx context.Context) {
	select {
	case <-ctx.Done():
		panic(ctx.Err())
	default:
	}
	i, _ := ctx.Value(tickCounter{}).(*int64)
	if i != nil {
		*i--
		if *i <= 0 {
			panic("exceeded execution limit")
		}
	}
}
