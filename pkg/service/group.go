package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type Service interface {
	Name() string
	Run(context.Context) error
}

type Group []Service

func (g Group) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var wg sync.WaitGroup
	errCh := make(chan error, len(g))
	wg.Add(len(g))
	for _, s := range g {
		go func(s Service) {
			defer wg.Done()
			if err := s.Run(runCtx); err != nil {
				errCh <- fmt.Errorf("%s: %w", s.Name(), err)
				cancelFn()
			}
		}(s)
	}

	<-runCtx.Done()
	wg.Wait()

	var err error
	close(errCh)
	for srvErr := range errCh {
		err = multierror.Append(err, srvErr)
	}
	return err
}
