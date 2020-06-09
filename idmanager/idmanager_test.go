package idmanager

import (
	"context"
	"reflect"
	"testing"

	"github.com/Scalingo/sand/store"
	"github.com/Scalingo/sand/store/storemock"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Generate(t *testing.T) {
	examples := map[string]struct {
		items    []int
		expected []int
		err      string
	}{
		"when no item, it should return 1, then 2": {
			items:    []int{},
			expected: []int{1, 2},
		},
		"when 1 is present, it should return 2, then 3": {
			items:    []int{1},
			expected: []int{2, 3},
		},
		"when 2 and 3 are present, it should return 1, then 4": {
			items:    []int{2, 3},
			expected: []int{1, 4},
		},
		"when 1 is present twice (data anomaly), it shouhld return 2, then 3": {
			items:    []int{1, 1},
			expected: []int{2, 3},
		},
		"when listing fails, it should return an error": {
			err: store.ErrNotFound.Error(),
		},
	}

	ctx := context.Background()
	for title, e := range examples {
		t.Run(title, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := storemock.NewMockStore(ctrl)

			manager := &manager{
				field:  "value",
				prefix: "/test-id",
				store:  store,
			}

			var items []map[string]interface{}
			if e.items != nil {
				for _, i := range e.items {
					items = append(items, map[string]interface{}{manager.field: float64(i)})
				}
			}
			stub := store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
				func(_ context.Context, _ string, _ bool, ptritems interface{}) {
					reflect.ValueOf(ptritems).Elem().Set(reflect.ValueOf(items))
				},
			)
			if e.err != "" {
				stub.Return(errors.New(e.err))
			}

			lock, err := manager.Lock(ctx)
			require.NoError(t, err)

			id, err := manager.Generate(ctx)
			require.NoError(t, lock.Unlock(ctx))
			if e.err != "" {
				require.Error(t, err)
				return
			}
			require.Equal(t, e.expected[0], id)

			// Add to the mock result the newly generated ID to prevent generating it again
			items = append(items, map[string]interface{}{manager.field: float64(e.expected[0])})
			store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
				func(_ context.Context, _ string, _ bool, ptritems interface{}) {
					reflect.ValueOf(ptritems).Elem().Set(reflect.ValueOf(items))
				},
			)
			id, err = manager.Generate(ctx)
			require.NoError(t, err)
			assert.Equal(t, e.expected[1], id)
		})
	}
}
