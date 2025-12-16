package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gamassss/url-shortener/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	HOT_COUNT  = 100
	WARM_COUNT = 10000
	COLD_COUNT = 9890000

	BATCH_SIZE  = 5000
	NUM_WORKERS = 4
)

type DataGenerator struct {
	pool *pgxpool.Pool
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbURL := cfg.Database.URL

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	gen := &DataGenerator{pool: pool}

	if err := gen.createTable(ctx); err != nil {
		log.Fatalf("Failed to create table: %v\n", err)
	}

	if err := gen.clearData(ctx); err != nil {
		log.Fatalf("Failed to clear data: %v\n", err)
	}

	if err := gen.insertHotURLs(ctx); err != nil {
		log.Fatalf("Failed to insert hot URLs: %v\n", err)
	}

	if err := gen.insertWarmURLs(ctx); err != nil {
		log.Fatalf("Failed to insert warm URLs: %v\n", err)
	}

	if err := gen.insertColdURLsParallel(ctx); err != nil {
		log.Fatalf("Failed to insert cold URLs: %v\n", err)
	}

	if err := gen.createIndexes(ctx); err != nil {
		log.Fatalf("Failed to create indexes: %v\n", err)
	}

	if err := gen.verifyData(ctx); err != nil {
		log.Printf("Warning: Data verification failed: %v\n", err)
	}
}

func (g *DataGenerator) createTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id BIGSERIAL PRIMARY KEY,
		short_code VARCHAR(20) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	)`
	_, err := g.pool.Exec(ctx, query)
	return err
}

func (g *DataGenerator) clearData(ctx context.Context) error {
	_, err := g.pool.Exec(ctx, "TRUNCATE urls RESTART IDENTITY")
	return err
}

func (g *DataGenerator) insertHotURLs(ctx context.Context) error {
	batch := &pgx.Batch{}

	for i := 1; i <= HOT_COUNT; i++ {
		shortCode := fmt.Sprintf("hot_%06d", i)
		originalURL := fmt.Sprintf("https://youtube.com/watch?v=%06d", i)
		batch.Queue(
			"INSERT INTO urls (short_code, original_url, created_at) VALUES ($1, $2, $3)",
			shortCode, originalURL, time.Now().Add(-time.Duration(i)*time.Minute),
		)
	}

	br := g.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch exec failed: %w", err)
		}
	}

	return nil
}

func (g *DataGenerator) insertWarmURLs(ctx context.Context) error {
	inserted := 0

	for start := 1; start <= WARM_COUNT; start += BATCH_SIZE {
		end := start + BATCH_SIZE - 1
		if end > WARM_COUNT {
			end = WARM_COUNT
		}

		batch := &pgx.Batch{}
		for i := start; i <= end; i++ {
			shortCode := fmt.Sprintf("warm_%06d", i)
			originalURL := fmt.Sprintf("https://github.com/repo/%06d", i)
			batch.Queue(
				"INSERT INTO urls (short_code, original_url, created_at) VALUES ($1, $2, $3)",
				shortCode, originalURL, time.Now().Add(-time.Duration(i)*time.Hour),
			)
		}

		br := g.pool.SendBatch(ctx, batch)
		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return fmt.Errorf("batch exec failed: %w", err)
			}
		}
		br.Close()

		inserted += (end - start + 1)
	}

	return nil
}

func (g *DataGenerator) insertColdURLsParallel(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, NUM_WORKERS)
	progressChan := make(chan int, NUM_WORKERS)

	done := make(chan bool)
	go func() {
		total := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case count := <-progressChan:
				total += count
			case <-ticker.C:
				_ = float64(total) / float64(COLD_COUNT) * 100
			case <-done:
				return
			}
		}
	}()

	rowsPerWorker := COLD_COUNT / NUM_WORKERS

	for workerID := 0; workerID < NUM_WORKERS; workerID++ {
		wg.Add(1)

		start := workerID*rowsPerWorker + 1
		end := start + rowsPerWorker - 1
		if workerID == NUM_WORKERS-1 {
			end = COLD_COUNT
		}

		go func(id, start, end int) {
			defer wg.Done()

			if err := g.insertColdURLsBatch(ctx, start, end, progressChan); err != nil {
				errChan <- fmt.Errorf("worker %d failed: %w", id, err)
			}
		}(workerID, start, end)
	}

	wg.Wait()
	close(done)
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func (g *DataGenerator) insertColdURLsBatch(ctx context.Context, start, end int, progress chan<- int) error {
	for i := start; i <= end; i += BATCH_SIZE {
		batchEnd := i + BATCH_SIZE - 1
		if batchEnd > end {
			batchEnd = end
		}

		batch := &pgx.Batch{}
		for j := i; j <= batchEnd; j++ {
			shortCode := fmt.Sprintf("cold_%07d", j)
			originalURL := fmt.Sprintf("https://example.com/page/%07d", j)
			batch.Queue(
				"INSERT INTO urls (short_code, original_url, created_at) VALUES ($1, $2, $3)",
				shortCode, originalURL, time.Now().Add(-time.Duration(j)*time.Second),
			)
		}

		br := g.pool.SendBatch(ctx, batch)
		for k := 0; k < batch.Len(); k++ {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return fmt.Errorf("batch exec failed: %w", err)
			}
		}
		br.Close()

		progress <- (batchEnd - i + 1)
	}

	return nil
}

func (g *DataGenerator) createIndexes(ctx context.Context) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code)",
		"CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at)",
	}

	for _, query := range indexes {
		if _, err := g.pool.Exec(ctx, query); err != nil {
			return err
		}
	}

	if _, err := g.pool.Exec(ctx, "ANALYZE urls"); err != nil {
		return err
	}

	return nil
}

func (g *DataGenerator) verifyData(ctx context.Context) error {
	var count int64
	err := g.pool.QueryRow(ctx, "SELECT COUNT(*) FROM urls").Scan(&count)
	if err != nil {
		return err
	}

	expected := int64(HOT_COUNT + WARM_COUNT + COLD_COUNT)
	if count != expected {
		return fmt.Errorf("expected %d rows but got %d", expected, count)
	}

	return nil
}
