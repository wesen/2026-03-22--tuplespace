package migrations

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyFS(ctx context.Context, db *pgxpool.Pool, migrationsFS fs.FS) error {
	entries, err := listEntries(migrationsFS)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		body, err := fs.ReadFile(migrationsFS, entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := db.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func ListFS(migrationsFS fs.FS) ([]string, error) {
	entries, err := listEntries(migrationsFS)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func listEntries(migrationsFS fs.FS) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}
