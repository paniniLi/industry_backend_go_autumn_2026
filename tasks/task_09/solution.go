package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	/* валидация входных параметров */
	if workers < 1 {
		return nil, ErrInvalidWorkers
	} else if ctx.Err() != nil {
		return nil, ctx.Err()
	} else if len(in) == 0 {
		return []Result[R]{}, nil
	}

	var results = make([]Result[R], len(in))

	/* канал обмена сообщениями между горутиной по выполнению задач и горутиной по их распределению */
	jobs := make(chan int)
	var wg sync.WaitGroup

	/* запускаем workers */
	for range min(workers, len(in)) {
		wg.Go(func() {
			/* каждый worker может обрабатывать задачи из jobs по мере освобождения */
			for index := range jobs {
				var funResult, funError = fn(ctx, in[index])
				var result R
				if funError == nil {
					result = funResult
				}
				results[index] = Result[R]{Value: result, Err: funError}
			}
		})
	}

	/* выдаем задания worker-ам */
dispatch:
	for index := range in {
		/* ожидаем одно из двух событий: контекст отменен, либо worker готов принять новую задачу */
		select {
		case jobs <- index:
		case <-ctx.Done():
			break dispatch
		}
	}
	close(jobs)

	/* ожидаем завершения всех горутин, fn сама принудительно завершится в случае отмены контекста */
	wg.Wait()
	/* обработка принудительного прерывания контекста */
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
