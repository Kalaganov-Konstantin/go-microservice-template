package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TestEntity struct {
	ID   string
	Name string
}

func (e *TestEntity) GetID() string {
	return e.ID
}

type RepositoryTestSuite struct {
	suite.Suite
	repo *Repository[*TestEntity]
	ctx  context.Context
}

func (s *RepositoryTestSuite) SetupTest() {
	s.repo = New[*TestEntity]()
	s.ctx = context.Background()
}

func (s *RepositoryTestSuite) SetupSubTest() {
	s.repo = New[*TestEntity]()
	s.ctx = context.Background()
}

func (s *RepositoryTestSuite) createTestEntity(id, name string) *TestEntity {
	return &TestEntity{ID: id, Name: name}
}

func (s *RepositoryTestSuite) saveTestEntity(entity *TestEntity) {
	err := s.repo.Save(s.ctx, entity)
	s.Require().NoError(err)
}

func (s *RepositoryTestSuite) TestNew() {
	repo := New[*TestEntity]()

	s.Require().NotNil(repo, "New() should not return nil")
	s.Require().NotNil(repo.data, "Repository data should be initialized")
	s.Assert().Empty(repo.data, "Repository data should be empty initially")
	s.Assert().IsType(&Repository[*TestEntity]{}, repo)
}

func (s *RepositoryTestSuite) TestSave() {
	tests := []struct {
		name          string
		entity        *TestEntity
		setupRepo     func()
		expectedError error
		validateState func()
	}{
		{
			name:          "successful_save",
			entity:        s.createTestEntity("test-id", "Test Entity"),
			setupRepo:     func() {},
			expectedError: nil,
			validateState: func() {
				savedEntity, exists := s.repo.data["test-id"]
				s.Require().True(exists, "Entity should exist in repository")
				s.Assert().Equal("test-id", savedEntity.GetID())
				s.Assert().Equal("Test Entity", savedEntity.Name)
			},
		},
		{
			name:   "entity_already_exists",
			entity: s.createTestEntity("existing-id", "New Entity"),
			setupRepo: func() {
				existingEntity := s.createTestEntity("existing-id", "Existing Entity")
				s.saveTestEntity(existingEntity)
			},
			expectedError: ErrAlreadyExists,
			validateState: func() {

				savedEntity, exists := s.repo.data["existing-id"]
				s.Require().True(exists)
				s.Assert().Equal("Existing Entity", savedEntity.Name)
			},
		},
		{
			name:          "save_with_empty_id",
			entity:        s.createTestEntity("", "Empty ID Entity"),
			setupRepo:     func() {},
			expectedError: nil,
			validateState: func() {
				_, exists := s.repo.data[""]
				s.Assert().True(exists)
			},
		},
		{
			name:          "save_with_unicode_characters",
			entity:        s.createTestEntity("unicode-тест-🌟", "Unicode Entity 测试"),
			setupRepo:     func() {},
			expectedError: nil,
			validateState: func() {
				savedEntity, exists := s.repo.data["unicode-тест-🌟"]
				s.Require().True(exists)
				s.Assert().Equal("Unicode Entity 测试", savedEntity.Name)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupRepo()
			err := s.repo.Save(s.ctx, tt.entity)

			if tt.expectedError != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedError)
			} else {
				s.Require().NoError(err)
			}

			if tt.validateState != nil {
				tt.validateState()
			}
		})
	}
}

func (s *RepositoryTestSuite) TestGetByID() {
	tests := []struct {
		name           string
		entityID       string
		setupRepo      func()
		expectedEntity *TestEntity
		expectedError  error
	}{
		{
			name:     "successful_get",
			entityID: "test-id",
			setupRepo: func() {
				entity := s.createTestEntity("test-id", "Test Entity")
				s.saveTestEntity(entity)
			},
			expectedEntity: s.createTestEntity("test-id", "Test Entity"),
			expectedError:  nil,
		},
		{
			name:           "entity_not_found",
			entityID:       "nonexistent-id",
			setupRepo:      func() {},
			expectedEntity: nil,
			expectedError:  ErrNotFound,
		},
		{
			name:     "get_with_empty_id",
			entityID: "",
			setupRepo: func() {
				entity := s.createTestEntity("", "Empty ID Entity")
				s.saveTestEntity(entity)
			},
			expectedEntity: s.createTestEntity("", "Empty ID Entity"),
			expectedError:  nil,
		},
		{
			name:     "get_with_unicode_id",
			entityID: "unicode-тест-🌟",
			setupRepo: func() {
				entity := s.createTestEntity("unicode-тест-🌟", "Unicode Entity")
				s.saveTestEntity(entity)
			},
			expectedEntity: s.createTestEntity("unicode-тест-🌟", "Unicode Entity"),
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupRepo()
			entity, err := s.repo.GetByID(s.ctx, tt.entityID)

			if tt.expectedError != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedError)
				s.Assert().Nil(entity)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(entity)
				s.Assert().Equal(tt.expectedEntity.ID, entity.ID)
				s.Assert().Equal(tt.expectedEntity.Name, entity.Name)
			}
		})
	}
}

func (s *RepositoryTestSuite) TestUpdate() {
	tests := []struct {
		name          string
		entity        *TestEntity
		setupRepo     func()
		expectedError error
		validateState func()
	}{
		{
			name:   "successful_update",
			entity: s.createTestEntity("test-id", "Updated Entity"),
			setupRepo: func() {
				originalEntity := s.createTestEntity("test-id", "Original Entity")
				s.saveTestEntity(originalEntity)
			},
			expectedError: nil,
			validateState: func() {
				updatedEntity, err := s.repo.GetByID(s.ctx, "test-id")
				s.Require().NoError(err)
				s.Assert().Equal("Updated Entity", updatedEntity.Name)
			},
		},
		{
			name:          "entity_not_found",
			entity:        s.createTestEntity("nonexistent-id", "Some Entity"),
			setupRepo:     func() {},
			expectedError: ErrNotFound,
			validateState: func() {
				count, err := s.repo.Count(s.ctx)
				s.Require().NoError(err)
				s.Assert().Equal(0, count)
			},
		},
		{
			name:   "update_preserves_other_entities",
			entity: s.createTestEntity("entity-1", "Updated Entity 1"),
			setupRepo: func() {
				s.saveTestEntity(s.createTestEntity("entity-1", "Original Entity 1"))
				s.saveTestEntity(s.createTestEntity("entity-2", "Entity 2"))
			},
			expectedError: nil,
			validateState: func() {
				entity1, err := s.repo.GetByID(s.ctx, "entity-1")
				s.Require().NoError(err)
				s.Assert().Equal("Updated Entity 1", entity1.Name)

				entity2, err := s.repo.GetByID(s.ctx, "entity-2")
				s.Require().NoError(err)
				s.Assert().Equal("Entity 2", entity2.Name)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupRepo()
			err := s.repo.Update(s.ctx, tt.entity)

			if tt.expectedError != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedError)
			} else {
				s.Require().NoError(err)
			}

			if tt.validateState != nil {
				tt.validateState()
			}
		})
	}
}

func (s *RepositoryTestSuite) TestDelete() {
	tests := []struct {
		name          string
		entityID      string
		setupRepo     func()
		expectedError error
		validateState func()
	}{
		{
			name:     "successful_delete",
			entityID: "test-id",
			setupRepo: func() {
				entity := s.createTestEntity("test-id", "Test Entity")
				s.saveTestEntity(entity)
			},
			expectedError: nil,
			validateState: func() {
				_, err := s.repo.GetByID(s.ctx, "test-id")
				s.Require().Error(err)
				s.Assert().ErrorIs(err, ErrNotFound)
			},
		},
		{
			name:          "entity_not_found",
			entityID:      "nonexistent-id",
			setupRepo:     func() {},
			expectedError: ErrNotFound,
			validateState: func() {
				count, err := s.repo.Count(s.ctx)
				s.Require().NoError(err)
				s.Assert().Equal(0, count)
			},
		},
		{
			name:     "delete_preserves_other_entities",
			entityID: "entity-1",
			setupRepo: func() {
				s.saveTestEntity(s.createTestEntity("entity-1", "Entity 1"))
				s.saveTestEntity(s.createTestEntity("entity-2", "Entity 2"))
				s.saveTestEntity(s.createTestEntity("entity-3", "Entity 3"))
			},
			expectedError: nil,
			validateState: func() {
				_, err := s.repo.GetByID(s.ctx, "entity-1")
				s.Require().Error(err)
				s.Assert().ErrorIs(err, ErrNotFound)

				entity2, err := s.repo.GetByID(s.ctx, "entity-2")
				s.Require().NoError(err)
				s.Assert().Equal("Entity 2", entity2.Name)

				entity3, err := s.repo.GetByID(s.ctx, "entity-3")
				s.Require().NoError(err)
				s.Assert().Equal("Entity 3", entity3.Name)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupRepo()
			err := s.repo.Delete(s.ctx, tt.entityID)

			if tt.expectedError != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedError)
			} else {
				s.Require().NoError(err)
			}

			if tt.validateState != nil {
				tt.validateState()
			}
		})
	}
}

func (s *RepositoryTestSuite) TestList() {
	s.Run("empty_repository", func() {
		entities, err := s.repo.List(s.ctx)

		s.Require().NoError(err)
		s.Assert().Empty(entities)
	})

	s.Run("repository_with_entities", func() {
		testEntities := []*TestEntity{
			s.createTestEntity("id1", "Entity 1"),
			s.createTestEntity("id2", "Entity 2"),
			s.createTestEntity("id3", "Entity 3"),
		}

		for _, entity := range testEntities {
			s.saveTestEntity(entity)
		}

		entities, err := s.repo.List(s.ctx)

		s.Require().NoError(err)
		s.Assert().Len(entities, len(testEntities))

		entityIDs := make(map[string]bool)
		for _, entity := range entities {
			entityIDs[entity.GetID()] = true
		}

		for _, expectedEntity := range testEntities {
			s.Assert().True(entityIDs[expectedEntity.ID], "Entity %s should be in the list", expectedEntity.ID)
		}
	})

	s.Run("list_with_large_dataset", func() {
		const numEntities = 1000
		for i := 0; i < numEntities; i++ {
			entity := s.createTestEntity(fmt.Sprintf("id-%d", i), fmt.Sprintf("Entity %d", i))
			s.saveTestEntity(entity)
		}

		entities, err := s.repo.List(s.ctx)

		s.Require().NoError(err)
		s.Assert().Len(entities, numEntities)
	})
}

func (s *RepositoryTestSuite) TestCount() {
	s.Run("empty_repository", func() {
		count, err := s.repo.Count(s.ctx)

		s.Require().NoError(err)
		s.Assert().Equal(0, count)
	})

	s.Run("repository_with_entities", func() {
		for i := 1; i <= 5; i++ {
			entity := s.createTestEntity(fmt.Sprintf("id%d", i), fmt.Sprintf("Entity %d", i))
			s.saveTestEntity(entity)
		}

		count, err := s.repo.Count(s.ctx)

		s.Require().NoError(err)
		s.Assert().Equal(5, count)
	})

	s.Run("count_after_operations", func() {

		for i := 1; i <= 10; i++ {
			entity := s.createTestEntity(fmt.Sprintf("id%d", i), fmt.Sprintf("Entity %d", i))
			s.saveTestEntity(entity)
		}

		count, err := s.repo.Count(s.ctx)
		s.Require().NoError(err)
		s.Assert().Equal(10, count)

		err = s.repo.Delete(s.ctx, "id1")
		s.Require().NoError(err)
		err = s.repo.Delete(s.ctx, "id5")
		s.Require().NoError(err)

		count, err = s.repo.Count(s.ctx)
		s.Require().NoError(err)
		s.Assert().Equal(8, count)
	})
}

func (s *RepositoryTestSuite) TestConcurrentAccess() {
	s.Run("concurrent_save_operations", func() {
		const numGoroutines = 10
		const entitiesPerGoroutine = 10

		errChan := make(chan error, numGoroutines*entitiesPerGoroutine)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < entitiesPerGoroutine; j++ {
					entity := s.createTestEntity(
						fmt.Sprintf("goroutine-%d-entity-%d", goroutineID, j),
						fmt.Sprintf("Entity %d-%d", goroutineID, j),
					)
					err := s.repo.Save(s.ctx, entity)
					errChan <- err
				}
			}(i)
		}

		go func() {
			wg.Wait()
			close(errChan)
		}()

		for err := range errChan {
			s.Assert().NoError(err, "Concurrent save operations should not fail")
		}

		count, err := s.repo.Count(s.ctx)
		s.Require().NoError(err)
		s.Assert().Equal(numGoroutines*entitiesPerGoroutine, count)
	})

	s.Run("concurrent_mixed_operations", func() {
		const numOperations = 100

		for i := 0; i < 50; i++ {
			entity := s.createTestEntity(fmt.Sprintf("pre-id-%d", i), fmt.Sprintf("Pre Entity %d", i))
			s.saveTestEntity(entity)
		}

		var wg sync.WaitGroup
		errorChan := make(chan error, numOperations*3)

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				entity := s.createTestEntity(fmt.Sprintf("save-id-%d", id), fmt.Sprintf("Save Entity %d", id))
				err := s.repo.Save(s.ctx, entity)
				errorChan <- err
			}(i)
		}

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := s.repo.GetByID(s.ctx, fmt.Sprintf("pre-id-%d", id%50))
				errorChan <- err
			}(i)
		}

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.repo.Count(s.ctx)
				errorChan <- err
			}()
		}

		go func() {
			wg.Wait()
			close(errorChan)
		}()

		for err := range errorChan {
			s.Assert().NoError(err, "Concurrent mixed operations should not fail")
		}
	})
}

func BenchmarkRepository_Save(b *testing.B) {
	repo := New[*TestEntity]()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entity := &TestEntity{
			ID:   fmt.Sprintf("bench-id-%d", i),
			Name: fmt.Sprintf("Bench Entity %d", i),
		}
		err := repo.Save(ctx, entity)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_GetByID(b *testing.B) {
	repo := New[*TestEntity]()
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		entity := &TestEntity{
			ID:   fmt.Sprintf("bench-id-%d", i),
			Name: fmt.Sprintf("Bench Entity %d", i),
		}
		_ = repo.Save(ctx, entity)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("bench-id-%d", i%1000)
		_, err := repo.GetByID(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_ConcurrentAccess(b *testing.B) {
	repo := New[*TestEntity]()
	ctx := context.Background()
	var counter int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := atomic.AddInt64(&counter, 1)
			entity := &TestEntity{
				ID:   fmt.Sprintf("concurrent-bench-id-%d", id),
				Name: fmt.Sprintf("Concurrent Bench Entity %d", id),
			}
			err := repo.Save(ctx, entity)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func (s *RepositoryTestSuite) TestContextTimeout() {
	s.Run("save_with_timeout", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(1 * time.Millisecond)

		entity := s.createTestEntity("timeout-test", "Timeout Entity")
		err := s.repo.Save(ctx, entity)

		s.Assert().NoError(err)
	})
}

func (s *RepositoryTestSuite) TestEdgeCases() {
	s.Run("nil_entity_handling", func() {

		zeroEntity := &TestEntity{}
		err := s.repo.Save(s.ctx, zeroEntity)
		s.Assert().NoError(err)

		retrievedEntity, err := s.repo.GetByID(s.ctx, "")
		s.Require().NoError(err)
		s.Assert().Equal(zeroEntity.Name, retrievedEntity.Name)
	})

	s.Run("very_long_id", func() {
		longIDBytes := make([]byte, 10000)
		for i := range longIDBytes {
			longIDBytes[i] = 'a'
		}
		longID := string(longIDBytes)

		entity := s.createTestEntity(longID, "Long ID Entity")
		s.saveTestEntity(entity)

		retrievedEntity, err := s.repo.GetByID(s.ctx, longID)
		s.Require().NoError(err)
		s.Assert().Equal("Long ID Entity", retrievedEntity.Name)
	})
}

func TestRepository_MemoryLeaks(t *testing.T) {
	repo := New[*TestEntity]()
	ctx := context.Background()

	const iterations = 10000

	for i := 0; i < iterations; i++ {
		entity := &TestEntity{
			ID:   fmt.Sprintf("leak-test-%d", i),
			Name: fmt.Sprintf("Leak Test Entity %d", i),
		}

		err := repo.Save(ctx, entity)
		if err != nil {
			t.Fatal(err)
		}

		err = repo.Delete(ctx, entity.ID)
		if err != nil {
			t.Fatal(err)
		}
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Errorf("Expected empty repository, got %d entities", count)
	}
}

func TestRepository_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large payload test in short mode")
	}

	repo := New[*TestEntity]()
	ctx := context.Background()

	largeDataBytes := make([]byte, 1024*1024) // 1MB
	for i := range largeDataBytes {
		largeDataBytes[i] = 'x'
	}
	largeData := string(largeDataBytes)

	entity := &TestEntity{
		ID:   "large-payload-test",
		Name: largeData,
	}

	err := repo.Save(ctx, entity)
	if err != nil {
		t.Fatal(err)
	}

	retrievedEntity, err := repo.GetByID(ctx, "large-payload-test")
	if err != nil {
		t.Fatal(err)
	}

	if len(retrievedEntity.Name) != len(largeData) {
		t.Errorf("Expected name length %d, got %d", len(largeData), len(retrievedEntity.Name))
	}

	err = repo.Delete(ctx, "large-payload-test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
