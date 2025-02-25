package idmanager

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/store/storemock"
)

func TestManager_Generate(t *testing.T) {
	t.Run("it should return the next available ID", func(t *testing.T) {
		// Given
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		existedIDs := []int{1, 2}
		expectedID := 3

		store := storemock.NewMockStore(ctrl)
		manager := &manager{
			field:  "value",
			prefix: "/test-id",
			store:  store,
			config: &config.Config{
				MaxVNI: 5,
			},
		}

		store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
			func(_ context.Context, _ string, _ bool, rawPtrItems interface{}) {
				ptrItems, ok := rawPtrItems.(*[]map[string]interface{})
				require.True(t, ok)
				for _, id := range existedIDs {
					*ptrItems = append(*ptrItems, map[string]interface{}{manager.field: float64(id)})
				}
			},
		).Return(nil)

		// When
		newID, err := manager.Generate(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedID, newID)
	})
	t.Run("it should return the next available ID even if there is an anomaly", func(t *testing.T) {
		// Given
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		existedIDs := []int{1, 1}
		expectedID := 2

		store := storemock.NewMockStore(ctrl)
		manager := &manager{
			field:  "value",
			prefix: "/test-id",
			store:  store,
			config: &config.Config{
				MaxVNI: 5,
			},
		}

		store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
			func(_ context.Context, _ string, _ bool, rawPtrItems interface{}) {
				ptrItems, ok := rawPtrItems.(*[]map[string]interface{})
				require.True(t, ok)
				for _, id := range existedIDs {
					*ptrItems = append(*ptrItems, map[string]interface{}{manager.field: float64(id)})
				}
			},
		).Return(nil)

		// When
		newID, err := manager.Generate(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedID, newID)
	})
	t.Run("it should return an error if the store fails to list the items", func(t *testing.T) {
		// Given
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		store := storemock.NewMockStore(ctrl)
		manager := &manager{
			field:  "value",
			prefix: "/test-id",
			store:  store,
			config: &config.Config{
				MaxVNI: 5,
			},
		}

		store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Return(assert.AnError)

		// When
		newID, err := manager.Generate(ctx)

		// Then
		require.Equal(t, -1, newID)
		require.Error(t, err)
		require.ErrorContains(t, err, "fail to get list of items")
	})
	t.Run("maxVNI should be allocable", func(t *testing.T) {
		// Given
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		existedIDs := []int{1, 2, 3, 4}
		expectedID := 5

		store := storemock.NewMockStore(ctrl)
		manager := &manager{
			field:  "value",
			prefix: "/test-id",
			store:  store,
			config: &config.Config{
				MaxVNI: 5,
			},
		}

		store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
			func(_ context.Context, _ string, _ bool, rawPtrItems interface{}) {
				ptrItems, ok := rawPtrItems.(*[]map[string]interface{})
				require.True(t, ok)
				for _, id := range existedIDs {
					*ptrItems = append(*ptrItems, map[string]interface{}{manager.field: float64(id)})
				}
			},
		).Return(nil)

		// When
		newID, err := manager.Generate(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedID, newID)
	})
	t.Run("it should return an error if there are no more available IDs", func(t *testing.T) {
		// Given
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		existedIDs := []int{1, 2, 3, 4, 5}

		store := storemock.NewMockStore(ctrl)
		manager := &manager{
			field:  "value",
			prefix: "/test-id",
			store:  store,
			config: &config.Config{
				MaxVNI: 5,
			},
		}

		store.EXPECT().Get(gomock.Any(), manager.prefix, true, gomock.Any()).Do(
			func(_ context.Context, _ string, _ bool, rawPtrItems interface{}) {
				ptrItems, ok := rawPtrItems.(*[]map[string]interface{})
				require.True(t, ok)
				for _, id := range existedIDs {
					*ptrItems = append(*ptrItems, map[string]interface{}{manager.field: float64(id)})
				}
			},
		).Return(nil)

		// When
		newID, err := manager.Generate(ctx)

		// Then
		require.Equal(t, -1, newID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoIDAvailable)
	})
}
