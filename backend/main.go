package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// --- ESTRUCTURAS ---
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Idea struct {
	ID            string `json:"ID"`
	TeamID        string `json:"TeamID"`
	Autor_id      string `json:"Autor_Id"`
	RolAutor      string `json:"RolAutor"`
	Text          string `json:"Text"`
	Type          string `json:"Type"`
	Status        string `json:"estado_colectivo"`
	ScoreApoyo    int    `json:"score_apoyo"`
	ScoreObjecion int    `json:"score_objecion"`
}

type Relation struct {
	TeamID string `json:"TeamID"`
	FromID string `json:"FromID"`
	ToID   string `json:"ToID"`
	Type   string `json:"Type"`
}

type Interaction struct {
	IdeaID     string `json:"IdeaID"`
	UserID     string `json:"UserID"`
	Type       string `json:"Type"`
	Intensidad int    `json:"Intensidad"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./cmp.db")
	if err != nil {
		log.Fatal(err)
	}

	// 1. Equipos
	db.Exec(`CREATE TABLE IF NOT EXISTS teams (id TEXT PRIMARY KEY, name TEXT);`)

	// 2. Ideas
	db.Exec(`CREATE TABLE IF NOT EXISTS ideas (
		id TEXT PRIMARY KEY,
		team_id TEXT,
		autor_id TEXT,
		rol_autor TEXT,
		text TEXT,
		type TEXT,
		status TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(team_id) REFERENCES teams(id)
	);`)

	// 3. Relaciones
	db.Exec(`CREATE TABLE IF NOT EXISTS relations (
		team_id TEXT, from_id TEXT, to_id TEXT, type TEXT,
		PRIMARY KEY (from_id, to_id), FOREIGN KEY(team_id) REFERENCES teams(id)
	);`)

	// 4. INTERACCIONES (CORREGIDO)
	// Eliminamos 'type' del UNIQUE. Ahora es solo (idea_id, user_id).
	// Esto obliga a que solo exista 1 fila por usuario por idea.
	db.Exec(`CREATE TABLE IF NOT EXISTS interactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		idea_id TEXT, user_id TEXT, type TEXT, intensidad INTEGER,
		CONSTRAINT unique_vote UNIQUE(idea_id, user_id)
	);`)

	log.Println("✅ Servidor CMP 0.2 (Voto Exclusivo) Listo.")
}

func calcularEstado(apoyo, objecion int, estadoDB string) string {
	if estadoDB == "finalizada" {
		return "finalizada"
	}
	umbralRelevancia := 4
	umbralConflicto := 3

	if apoyo < umbralRelevancia && objecion < umbralConflicto {
		return "emergente"
	}
	if apoyo >= umbralRelevancia && objecion >= umbralConflicto {
		return "controvertida"
	}
	if apoyo < umbralRelevancia && objecion >= umbralConflicto {
		return "en_riesgo"
	}
	if apoyo >= umbralRelevancia && objecion < umbralConflicto {
		return "casi_consensuada"
	}
	return "en_debate"
}

// --- HANDLERS ---

func ideasHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method == "GET" {
		team := r.URL.Query().Get("team")
		var check string
		if err := db.QueryRow("SELECT id FROM teams WHERE id = ?", team).Scan(&check); err != nil {
			http.Error(w, "Equipo no encontrado", 404)
			return
		}

		rows, err := db.Query("SELECT id, autor_id, COALESCE(rol_autor, 'participante'), text, type, COALESCE(status, 'emergente') FROM ideas WHERE team_id = ?", team)
		if err != nil {
			http.Error(w, "Error DB", 500)
			return
		}
		defer rows.Close()

		var list []Idea
		for rows.Next() {
			var i Idea
			var dbStatus string
			rows.Scan(&i.ID, &i.Autor_id, &i.RolAutor, &i.Text, &i.Type, &dbStatus)

			db.QueryRow("SELECT COALESCE(SUM(intensidad), 0) FROM interactions WHERE idea_id = ? AND type = 'apoyo'", i.ID).Scan(&i.ScoreApoyo)
			db.QueryRow("SELECT COALESCE(SUM(intensidad), 0) FROM interactions WHERE idea_id = ? AND type = 'objecion'", i.ID).Scan(&i.ScoreObjecion)

			i.Status = calcularEstado(i.ScoreApoyo, i.ScoreObjecion, dbStatus)
			list = append(list, i)
		}
		json.NewEncoder(w).Encode(list)
	}

	if r.Method == "POST" {
		var i Idea
		if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
			http.Error(w, "JSON Inválido", 400)
			return
		}
		_, err := db.Exec("INSERT INTO ideas (id, team_id, autor_id, rol_autor, text, type, status) VALUES (?, ?, ?, ?, ?, ?, ?)",
			i.ID, i.TeamID, i.Autor_id, i.RolAutor, i.Text, i.Type, "emergente")
		if err != nil {
			log.Println("Error insertando idea:", err)
			http.Error(w, "Error DB Insert", 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func interactHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method == "POST" {
		var voto Interaction
		json.NewDecoder(r.Body).Decode(&voto)

		// LÓGICA DE INTERCAMBIO DE VOTO (UPSERT ACTUALIZADO)
		// Si el usuario ya votó (aunque sea apoyo), y ahora vota objeción,
		// SQLite detecta conflicto en (idea_id, user_id).
		// Entonces actualiza el TYPE y la INTENSIDAD.
		query := `INSERT INTO interactions (idea_id, user_id, type, intensidad) VALUES (?, ?, ?, ?)
		ON CONFLICT(idea_id, user_id) 
		DO UPDATE SET type=excluded.type, intensidad=excluded.intensidad;`

		_, err := db.Exec(query, voto.IdeaID, voto.UserID, voto.Type, voto.Intensidad)
		if err != nil {
			log.Println("Error votando:", err) // Debug
			http.Error(w, "Error voting", 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// ... Resto de handlers (relations, team, resolve) sin cambios ...
func relationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method == "GET" {
		team := r.URL.Query().Get("team")
		rows, _ := db.Query("SELECT from_id, to_id, type FROM relations WHERE team_id = ?", team)
		defer rows.Close()
		var list []Relation
		for rows.Next() {
			var r Relation
			rows.Scan(&r.FromID, &r.ToID, &r.Type)
			list = append(list, r)
		}
		if list == nil {
			list = []Relation{}
		}
		json.NewEncoder(w).Encode(list)
	}
	if r.Method == "POST" {
		var rel Relation
		json.NewDecoder(r.Body).Decode(&rel)
		db.Exec("INSERT INTO relations (team_id, from_id, to_id, type) VALUES (?, ?, ?, ?)", rel.TeamID, rel.FromID, rel.ToID, rel.Type)
		w.WriteHeader(http.StatusCreated)
	}
}

func createTeamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method == "POST" {
		var t Team
		json.NewDecoder(r.Body).Decode(&t)
		t.ID = strings.ToLower(strings.TrimSpace(t.ID))
		var exists string
		err := db.QueryRow("SELECT id FROM teams WHERE id = ?", t.ID).Scan(&exists)
		if err == nil {
			http.Error(w, "Existe", 409)
			return
		}
		db.Exec("INSERT INTO teams (id, name) VALUES (?, ?)", t.ID, t.Name)
		w.WriteHeader(http.StatusCreated)
	}
}

func checkTeamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method == "GET" {
		id := r.URL.Query().Get("id")
		var name string
		err := db.QueryRow("SELECT name FROM teams WHERE id = ?", id).Scan(&name)
		if err != nil {
			http.Error(w, "No existe", 404)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": id, "name": name})
	}
}

func resolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method == "POST" {
		var req struct{ IdeaID string }
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("UPDATE ideas SET status = 'finalizada' WHERE id = ?", req.IdeaID)
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	initDB()
	http.HandleFunc("/", ideasHandler)
	http.HandleFunc("/connect", relationsHandler)
	http.HandleFunc("/interact", interactHandler)
	http.HandleFunc("/resolve", resolveHandler)
	http.HandleFunc("/create-team", createTeamHandler)
	http.HandleFunc("/check-team", checkTeamHandler)
	log.Println("🚀 Servidor CMP 0.2 (Voto Exclusivo) Corriendo en :8080")
	http.ListenAndServe(":8080", nil)
}
