package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"reesource-tracker/lib/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	confirm := flag.String("confirm", "", "must be RESET-ADMINS to execute fallback")
	flag.Parse()

	if *confirm != "RESET-ADMINS" {
		fmt.Println("refusing to run: pass --confirm RESET-ADMINS")
		os.Exit(2)
	}

	ctx := context.Background()
	if err := database.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer database.Disconnect()

	tx, err := database.Instance.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	q := database.New(tx)
	if err := q.EnableAdminRemovalOverride(ctx); err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}
	if err := q.ClearAllAdminRoles(ctx); err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]any{
		"source": "fallback_script",
		"ran_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := q.InsertAuditLog(ctx, database.InsertAuditLogParams{
		ActorUserID: sql.Null[[]byte]{Valid: false},
		Action:      "fallback_triggered",
		TargetType:  "system",
		TargetID:    sql.NullString{String: "admin_roles", Valid: true},
		Metadata:    payload,
	}); err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("fallback complete: all admin roles removed")
}
