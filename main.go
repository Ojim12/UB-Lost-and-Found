package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "ub_staff_session"

type Item struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Location    string `json:"location"`
	DateFound   string `json:"dateFound"`
	Status      string `json:"status"`
	Photo       string `json:"photo"`
}
type Claim struct {
    ID       string `json:"id"`
    Item     string `json:"item"`
    Person   string `json:"person"`
    Proof    string `json:"proof"`
    Status   string `json:"status"`
    ClaimDate string `json:"claimDate"`
}

type StaffSession struct {
	ID        string
	Name      string
	Role      string
	ExpiresAt time.Time
}

var (
	db       *sql.DB
	sessions = make(map[string]StaffSession)
	mu       sync.Mutex
)

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func createSessionToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func getSession(r *http.Request) (StaffSession, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return StaffSession{}, false
	}

	mu.Lock()
	defer mu.Unlock()

	session, exists := sessions[cookie.Value]

	if !exists {
		return StaffSession{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return StaffSession{}, false
	}

	return session, true
}

// POST /api/login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid login information.",
		})
		return
	}

	var staffID, staffName, role, passwordHash string

	err := db.QueryRow(`
		SELECT staff_id, staff_name, role, password_hash
		FROM staff
		WHERE email = ?
	`, credentials.Email).Scan(
		&staffID,
		&staffName,
		&role,
		&passwordHash,
	)

	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Incorrect email or password.",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(credentials.Password),
	)

	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Incorrect email or password.",
		})
		return
	}

	token, err := createSessionToken()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not create login session.",
		})
		return
	}

	session := StaffSession{
		ID:        staffID,
		Name:      staffName,
		Role:      role,
		ExpiresAt: time.Now().Add(8 * time.Hour),
	}

	mu.Lock()
	sessions[token] = session
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true when using HTTPS in production.
		MaxAge:   8 * 60 * 60,
	})

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Login successful.",
		"name":    staffName,
		"role":    role,
	})
}

// GET /api/me
func meHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
		return
	}

	session, authenticated := getSession(r)

	if !authenticated {
		sendJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Not logged in.",
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"name": session.Name,
		"role": session.Role,
	})
}

// POST /api/logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
		return
	}

	cookie, err := r.Cookie(sessionCookie)

	if err == nil {
		mu.Lock()
		delete(sessions, cookie.Value)
		mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully.",
	})
}

// POST /api/items - Only logged-in staff can add items.
func createItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
		return
	}

	session, authenticated := getSession(r)
	if !authenticated {
		sendJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Please log in as staff first.",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var input struct {
		Description string `json:"description"`
		Category    string `json:"category"`
		Location    string `json:"location"`
		DateFound   string `json:"dateFound"`
		FinderInfo  string `json:"finderInfo"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid form information.",
		})
		return
	}

	input.Description = strings.TrimSpace(input.Description)
	input.Category = strings.TrimSpace(input.Category)
	input.Location = strings.TrimSpace(input.Location)
	input.FinderInfo = strings.TrimSpace(input.FinderInfo)

	if len([]rune(input.Description)) < 10 ||
		len([]rune(input.Description)) > 500 ||
		input.Category == "" ||
		input.Location == "" ||
		input.FinderInfo == "" ||
		len([]rune(input.FinderInfo)) > 200 {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Please check the required fields and their lengths.",
		})
		return
	}

	foundDate, err := time.Parse("2006-01-02", input.DateFound)
	if err != nil || foundDate.After(time.Now().Truncate(24*time.Hour)) {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Please enter a valid date that is not in the future.",
		})
		return
	}

	var categoryID, locationID string

	err = db.QueryRow(
		"SELECT category_id FROM category WHERE category_name = ?",
		input.Category,
	).Scan(&categoryID)

	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Please select a valid category.",
		})
		return
	}

	err = db.QueryRow(
		"SELECT location_id FROM location WHERE location_name = ?",
		input.Location,
	).Scan(&locationID)

	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Please select a valid location.",
		})
		return
	}

	_, err = db.Exec(`
		INSERT INTO item (
			description,
			date_found,
			finder_info,
			status,
			category_id,
			location_id,
			logged_by_staff_id
		)
		VALUES (?, ?, ?, 'Unclaimed', ?, ?, ?)
	`,
		input.Description,
		input.DateFound,
		input.FinderInfo,
		categoryID,
		locationID,
		session.ID,
	)

	if err != nil {
		log.Println("Could not save item:", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not save the item.",
		})
		return
	}

	sendJSON(w, http.StatusCreated, map[string]string{
		"message": "Found item saved successfully.",
	})
}


// GET /api/items
func itemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
		return
	}

	rows, err := db.Query(`
		SELECT
			i.item_id,
			i.description,
			c.category_name,
			l.location_name,
			i.date_found,
			i.status,
			COALESCE(i.photo, '')
		FROM item i
		JOIN category c ON c.category_id = i.category_id
		JOIN location l ON l.location_id = i.location_id
		WHERE i.archived_at IS NULL
		ORDER BY i.date_found DESC
	`)

	if err != nil {
		log.Println("Item query error:", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not retrieve items.",
		})
		return
	}
	defer rows.Close()

	items := []Item{}

	for rows.Next() {
		var item Item
		var dateFound time.Time

		err := rows.Scan(
			&item.ID,
			&item.Description,
			&item.Category,
			&item.Location,
			&dateFound,
			&item.Status,
			&item.Photo,
		)

		if err != nil {
			log.Println("Item scan error:", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Could not read item information.",
			})
			return
		}

		item.DateFound = dateFound.Format("2006-01-02")
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Println("Item rows error:", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not read item information.",
		})
		return
	}

	sendJSON(w, http.StatusOK, items)
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
            "error": "Method not allowed.",
        })
        return
    }

    _, authenticated := getSession(r)
    if !authenticated {
        sendJSON(w, http.StatusUnauthorized, map[string]string{
            "error": "Please log in as staff first.",
        })
        return
    }

    var total, claimed, pending, approved, rejected int

    err := db.QueryRow(`
        SELECT
            COUNT(*),
            COALESCE(SUM(status = 'Claimed'), 0)
        FROM item
    `).Scan(&total, &claimed)

    if err != nil {
        log.Println("Report item query error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not retrieve inventory statistics.",
        })
        return
    }

    err = db.QueryRow(`
        SELECT
            COALESCE(SUM(claim_status = 'Pending'), 0),
            COALESCE(SUM(claim_status = 'Approved'), 0),
            COALESCE(SUM(claim_status = 'Rejected'), 0)
        FROM claim
    `).Scan(&pending, &approved, &rejected)

    if err != nil {
        log.Println("Report claim query error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not retrieve claim statistics.",
        })
        return
    }

    var successRate float64

    decidedClaims := approved + rejected
    if decidedClaims > 0 {
        successRate = float64(approved) / float64(decidedClaims) * 100
    }

    rows, err := db.Query(`
        SELECT c.category_name, COUNT(i.item_id)
        FROM category c
        LEFT JOIN item i ON i.category_id = c.category_id
        GROUP BY c.category_id, c.category_name
        ORDER BY c.category_name
    `)

    if err != nil {
        log.Println("Report category query error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not retrieve category statistics.",
        })
        return
    }
    defer rows.Close()

    type CategoryStat struct {
        Category string `json:"category"`
        Count    int    `json:"count"`
    }

    categories := []CategoryStat{}

    for rows.Next() {
        var category CategoryStat

        if err := rows.Scan(&category.Category, &category.Count); err != nil {
            log.Println("Category scan error:", err)
            sendJSON(w, http.StatusInternalServerError, map[string]string{
                "error": "Could not read category statistics.",
            })
            return
        }

        categories = append(categories, category)
    }

    if err := rows.Err(); err != nil {
        log.Println("Category rows error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not read category statistics.",
        })
        return
    }

    sendJSON(w, http.StatusOK, map[string]interface{}{
        "total":      total,
        "claimed":    claimed,
        "pending":    pending,
        "rate":       successRate,
        "categories": categories,
    })
}

func claimsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
            "error": "Method not allowed.",
        })
        return
    }

    _, authenticated := getSession(r)
    if !authenticated {
        sendJSON(w, http.StatusUnauthorized, map[string]string{
            "error": "Please log in as staff first.",
        })
        return
    }

    rows, err := db.Query(`
        SELECT
            c.claim_id,
            i.description,
            cl.claimant_name,
            c.proof_description,
            c.claim_status,
            c.claim_date
        FROM claim c
        JOIN item i ON i.item_id = c.item_id
        JOIN claimant cl ON cl.claimant_id = c.claimant_id
        ORDER BY c.claim_date DESC
    `)

    if err != nil {
        log.Println("Claim query error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not retrieve claims.",
        })
        return
    }
    defer rows.Close()

    claims := []Claim{}

    for rows.Next() {
        var claim Claim
        var claimDate time.Time

        err := rows.Scan(
            &claim.ID,
            &claim.Item,
            &claim.Person,
            &claim.Proof,
            &claim.Status,
            &claimDate,
        )

        if err != nil {
            log.Println("Claim scan error:", err)
            sendJSON(w, http.StatusInternalServerError, map[string]string{
                "error": "Could not read claim information.",
            })
            return
        }

        claim.ClaimDate = claimDate.Format("2006-01-02")
        claims = append(claims, claim)
    }

    if err := rows.Err(); err != nil {
        log.Println("Claim rows error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not read claim information.",
        })
        return
    }

    sendJSON(w, http.StatusOK, claims)
}
func claimDecisionHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
            "error": "Method not allowed.",
        })
        return
    }

    session, authenticated := getSession(r)
    if !authenticated {
        sendJSON(w, http.StatusUnauthorized, map[string]string{
            "error": "Please log in as staff first.",
        })
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 4096)

    var input struct {
        ClaimID  string `json:"claimId"`
        Decision string `json:"decision"`
        Notes    string `json:"notes"`
    }

    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Invalid claim information.",
        })
        return
    }

    input.ClaimID = strings.TrimSpace(input.ClaimID)
    input.Decision = strings.TrimSpace(input.Decision)
    input.Notes = strings.TrimSpace(input.Notes)

    if input.ClaimID == "" ||
        (input.Decision != "Approved" && input.Decision != "Rejected") ||
        input.Notes == "" ||
        len([]rune(input.Notes)) > 255 {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Please provide a valid decision and verification notes.",
        })
        return
    }

    tx, err := db.Begin()
    if err != nil {
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not process the claim.",
        })
        return
    }
    defer tx.Rollback()

    var itemID string
    var currentStatus string

    err = tx.QueryRow(`
        SELECT item_id, claim_status
        FROM claim
        WHERE claim_id = ?
        FOR UPDATE
    `, input.ClaimID).Scan(&itemID, &currentStatus)

    if err == sql.ErrNoRows {
        sendJSON(w, http.StatusNotFound, map[string]string{
            "error": "Claim not found.",
        })
        return
    }

    if err != nil {
        log.Println("Claim lookup error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not find the claim.",
        })
        return
    }

    if currentStatus != "Pending" {
        sendJSON(w, http.StatusConflict, map[string]string{
            "error": "This claim has already been decided.",
        })
        return
    }

    _, err = tx.Exec(`
        UPDATE claim
        SET claim_status = ?,
            verified_by_staff_id = ?,
            verification_decision = ?
        WHERE claim_id = ?
    `, input.Decision, session.ID, input.Notes, input.ClaimID)

    if err != nil {
        log.Println("Claim decision error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not save the claim decision.",
        })
        return
    }

    if input.Decision == "Approved" {
        _, err = tx.Exec(`
            UPDATE item
            SET status = 'Claimed'
            WHERE item_id = ?
        `, itemID)
    } else {
        _, err = tx.Exec(`
            UPDATE item
            SET status = 'Unclaimed'
            WHERE item_id = ?
              AND status = 'Pending Verification'
              AND NOT EXISTS (
                  SELECT 1 FROM claim
                  WHERE item_id = ?
                    AND claim_status = 'Pending'
              )
        `, itemID, itemID)
    }

    if err != nil {
        log.Println("Item status update error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not update the item status.",
        })
        return
    }

    if err := tx.Commit(); err != nil {
        log.Println("Claim transaction error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not complete the claim decision.",
        })
        return
    }

    sendJSON(w, http.StatusOK, map[string]string{
        "message": "Claim decision saved successfully.",
    })
}

func createClaimHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
            "error": "Method not allowed.",
        })
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 8192)

    var input struct {
        ItemID   string `json:"itemId"`
        Name     string `json:"name"`
        Type     string `json:"type"`
        UBID     string `json:"ubId"`
        Contact  string `json:"contact"`
        Proof    string `json:"proof"`
    }

    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Invalid claim information.",
        })
        return
    }

    input.ItemID = strings.TrimSpace(input.ItemID)
    input.Name = strings.TrimSpace(input.Name)
    input.Type = strings.TrimSpace(input.Type)
    input.UBID = strings.TrimSpace(input.UBID)
    input.Contact = strings.TrimSpace(input.Contact)
    input.Proof = strings.TrimSpace(input.Proof)

    if input.ItemID == "" ||
        input.Name == "" || len([]rune(input.Name)) > 100 ||
        (input.Type != "Student" && input.Type != "Faculty" && input.Type != "Staff") ||
        len(input.UBID) > 10 ||
        input.Contact == "" || len([]rune(input.Contact)) > 100 ||
        input.Proof == "" || len([]rune(input.Proof)) > 500 {

        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Please complete all required fields correctly.",
        })
        return
    }

    tx, err := db.Begin()
    if err != nil {
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not submit your claim.",
        })
        return
    }
    defer tx.Rollback()

    var itemStatus string

    err = tx.QueryRow(`
        SELECT status
        FROM item
        WHERE item_id = ?
        FOR UPDATE
    `, input.ItemID).Scan(&itemStatus)

    if err == sql.ErrNoRows {
        sendJSON(w, http.StatusNotFound, map[string]string{
            "error": "The selected item could not be found.",
        })
        return
    }

    if err != nil {
        log.Println("Claim item lookup error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not find the selected item.",
        })
        return
    }

    if itemStatus == "Claimed" {
        sendJSON(w, http.StatusConflict, map[string]string{
            "error": "This item has already been claimed.",
        })
        return
    }

    var claimantID string

    err = tx.QueryRow("SELECT UUID()").Scan(&claimantID)
    if err != nil {
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not create your claim.",
        })
        return
    }

    var ubID interface{}
    if input.UBID != "" {
        ubID = input.UBID
    }

    _, err = tx.Exec(`
        INSERT INTO claimant (
            claimant_id,
            claimant_name,
            claimant_type,
            ub_id_number,
            contact_info
        )
        VALUES (?, ?, ?, ?, ?)
    `, claimantID, input.Name, input.Type, ubID, input.Contact)

    if err != nil {
        log.Println("Claimant insert error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not save claimant information.",
        })
        return
    }

    _, err = tx.Exec(`
        INSERT INTO claim (
            item_id,
            claimant_id,
            claim_status,
            proof_description
        )
        VALUES (?, ?, 'Pending', ?)
    `, input.ItemID, claimantID, input.Proof)

    if err != nil {
        log.Println("Claim insert error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not save your claim.",
        })
        return
    }

    if itemStatus == "Unclaimed" {
        _, err = tx.Exec(`
            UPDATE item
            SET status = 'Pending Verification'
            WHERE item_id = ?
        `, input.ItemID)

        if err != nil {
            log.Println("Item status update error:", err)
            sendJSON(w, http.StatusInternalServerError, map[string]string{
                "error": "Could not update the item status.",
            })
            return
        }
    }

    if err := tx.Commit(); err != nil {
        log.Println("Claim transaction error:", err)
        sendJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Could not complete your claim submission.",
        })
        return
    }

    sendJSON(w, http.StatusCreated, map[string]string{
        "message": "Your claim has been submitted successfully and is pending staff verification.",
    })
}

func main() {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	if dbUser == "" || dbPassword == "" {
		log.Fatal("Please set DB_USER and DB_PASSWORD before starting the server.")
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3306)/ub_lost_found?parseTime=true",
		dbUser,
		dbPassword,
	)

	var err error

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Could not open database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Could not connect to MySQL:", err)
	}

	log.Println("Connected to the UB Lost and Found database!")

	
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/me", meHandler)
	http.HandleFunc("/api/logout", logoutHandler)
	http.HandleFunc("/api/claims", claimsHandler)
http.HandleFunc("/api/claims/decision", claimDecisionHandler)
http.HandleFunc("/api/claims/submit", createClaimHandler)
http.HandleFunc("/api/reports", reportsHandler)
	http.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        createItemHandler(w, r)
        return
    }

    itemsHandler(w, r)
})

	http.Handle("/", http.FileServer(http.Dir("public")))

	log.Println("Website running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}