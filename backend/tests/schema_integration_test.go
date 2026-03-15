//go:build integration

package tests

import (
	"context"
	"testing"
)

func TestStudySchemaTablesExist(t *testing.T) {
	ctx := context.Background()
	db := mustOpenDB(t)

	tables := []string{
		"studies",
		"groups",
		"source_items",
		"video_assets",
		"participants",
		"pair_presentations",
		"responses",
		"interaction_logs",
	}

	for _, tbl := range tables {
		var exists bool
		err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, tbl).Scan(&exists)
		if err != nil {
			t.Fatalf("query table %s: %v", tbl, err)
		}
		if !exists {
			t.Fatalf("expected table %s to exist", tbl)
		}
	}
}
