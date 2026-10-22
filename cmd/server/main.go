package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	khatabook "khata-book"
	"khata-book/handlers"
	"khata-book/models"
	"khata-book/repository"
	"khata-book/services"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "khata.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Connecting to SQLite database at: %s", dbPath)
	db, err := repository.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Initialize Go channel-based write queue to serialize write transactions
	writeQueue := repository.NewWriteQueue(db, 512)
	defer writeQueue.Close()

	// Initialize Repositories
	nodeRepo := repository.NewNodeRepository(db, writeQueue)
	entryRepo := repository.NewEntryRepository(db, writeQueue)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize Services
	khataService := services.NewKhataService(nodeRepo, entryRepo, analyticsRepo, userRepo)

	// Initialize Handlers & Routes
	router := handlers.NewRouter(khataService)
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Static Assets & UI Setup
	var staticFS http.FileSystem
	var indexHTML []byte

	if info, err := os.Stat("web"); err == nil && info.IsDir() {
		log.Println("Serving frontend assets directly from ./web directory")
		staticFS = http.Dir("web/static")
		indexHTML, err = os.ReadFile("web/index.html")
		if err != nil {
			log.Fatalf("Failed to read web/index.html: %v", err)
		}
	} else {
		log.Println("Serving frontend assets from embedded binary FS")
		subStatic, err := fs.Sub(khatabook.EmbeddedWebFS, "web/static")
		if err != nil {
			log.Fatalf("Failed to sub embedded static fs: %v", err)
		}
		staticFS = http.FS(subStatic)
		indexHTML, err = khatabook.EmbeddedWebFS.ReadFile("web/index.html")
		if err != nil {
			log.Fatalf("Failed to read embedded index.html: %v", err)
		}
	}

	// Serve Static Files with no-cache headers to prevent stale CSS/JS in browsers
	staticFileServer := http.StripPrefix("/static/", http.FileServer(staticFS))
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		staticFileServer.ServeHTTP(w, r)
	})

	// Serve Frontend Single-Page App Index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		if _, err := os.Stat("web/index.html"); err == nil {
			if content, err := os.ReadFile("web/index.html"); err == nil {
				w.Write(content)
				return
			}
		}
		w.Write(indexHTML)
	})

	// Wrap API with Authentication & Department Security Middleware
	appHandler := handlers.AuthMiddleware(khataService)(mux)

	addr := fmt.Sprintf(":%s", port)
	localURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("==================================================================")
	fmt.Println(" 📒 ENTERPRISE KHATA BOOK - Hierarchical Organization Ledger")
	fmt.Println("==================================================================")
	fmt.Printf(" [✓] Local URL: %s\n", localURL)
	fmt.Printf(" [✓] Database: %s (SQLite WAL Mode + Serialized Write Queue)\n", dbPath)
	fmt.Printf(" [✓] Max Depth Limit: %d Levels (Department -> Group -> Subgroups)\n", models.MaxDepth)
	fmt.Println(" [✓] Currency: Indian Rupees (₹) stored in zero-loss integer Paise")
	fmt.Println(" [✓] Concurrency: Go Channel-Based Worker Pool preventing DB locks")
	fmt.Println(" [✓] Pre-seeded RBAC Users:")
	fmt.Println("     - Alice Sharma (admin): Global Admin (All Departments)")
	fmt.Println("     - Bob Verma (finance_user): Scoped to Finance & Accounts")
	fmt.Println("     - Charlie Patel (ops_user): Scoped to Operations & Logistics")
	fmt.Println("     - Diana Rao (sales_user): Scoped to Sales & Distribution")
	fmt.Println("==================================================================")

	if err := http.ListenAndServe(addr, appHandler); err != nil {
		log.Fatalf("Server stopped unexpectedly: %v", err)
	}
}
