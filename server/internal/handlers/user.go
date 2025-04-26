
package handlers

import (
	"chat-app/internal/config"
	"chat-app/internal/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := config.DB.Query(`
		SELECT id, username, email, avatar, is_online, last_seen
		FROM users
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var avatar sql.NullString
		
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, 
			&avatar, &user.IsOnline, &user.LastSeen,
		)
		if err != nil {
			http.Error(w, "Error scanning users", http.StatusInternalServerError)
			return
		}

		if avatar.Valid {
			user.Avatar = avatar.String
		}

		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
