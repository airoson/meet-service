package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"meet-service/utils"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

var (
	emailReg = regexp.MustCompile(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)
	phoneReg = regexp.MustCompile(`^[\+]?[(]?[0-9]{3}[)]?[-\s\.]?[0-9]{3}[-\s\.]?[0-9]{4,6}$`)
)

func (restApi *RestApi) HandleSignupRequest(response http.ResponseWriter, request *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			response.WriteHeader(400)
			log.Printf("Error in signup handler: %v\n", err)
		}
	}()
	req := UserRequest{}
	data, err := io.ReadAll(request.Body)
	fatal(err)
	err = json.Unmarshal(data, &req)
	if err != nil {
		WriteMessage(response, "Wrong user data", 400)
		return
	}
	var credentialName string
	if phoneReg.Match([]byte(req.Identifier)) {
		credentialName = "phone"
	} else if emailReg.Match([]byte(req.Identifier)) {
		credentialName = "email"
	} else {
		WriteMessage(response, "Wrong identifier", 400)
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	fatal(err)
	userId := uuid.New().String()
	var ctx context.Context = context.Background()
	tx, err := restApi.Db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	fatal(err)
	rows, err := tx.QueryContext(ctx,
		`SELECT * from registered_user 
		WHERE email = $1 OR phone = $1;`, req.Identifier)
	if err != nil {
		tx.Rollback()
	}
	fatal(err)
	if rows.Next() {
		WriteMessage(response, fmt.Sprintf("Can't create user: %s is already taken", credentialName), 400)
		return
	}
	var email, phone sql.NullString
	if credentialName == "phone" {
		phone.String = req.Identifier
		phone.Valid = true
	} else {
		email.String = req.Identifier
		email.Valid = true
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO registered_user(user_id, phone, email, password, role) VALUES ($1, $2, $3, $4, 1);`, userId, phone, email, hashed)
	if err != nil {
		tx.Rollback()
	}
	fatal(err)
	tx.Commit()
	msg := struct {
		UserId string `json:"userId"`
	}{userId}
	data, _ = json.Marshal(msg)
	response.WriteHeader(200)
	response.Write(data)
}

func (restApi *RestApi) HandleLoginRequest(response http.ResponseWriter, request *http.Request) {
	req := UserRequest{}
	data, err := io.ReadAll(request.Body)
	fatal(err)
	err = json.Unmarshal(data, &req)
	if err != nil {
		WriteMessage(response, "Wrong body data", 400)
		return
	}
	var rows *sql.Rows
	if phoneReg.Match([]byte(req.Identifier)) {
		rows, err = restApi.Db.Query("SELECT user_id, password, name FROM registered_user JOIN roles ON role=role_id WHERE phone = $1;", req.Identifier)
	} else if emailReg.Match([]byte(req.Identifier)) {
		rows, err = restApi.Db.Query("SELECT user_id, password, name FROM registered_user JOIN roles ON role=role_id WHERE email = $1;", req.Identifier)
	}
	fatal(err)
	if !rows.Next() {
		log.Println("Can't find user with given credentials")
		WriteMessage(response, "Wrong user credentials", 401)
		return
	}
	var (
		password []byte
		userId   string
		role     string
	)
	rows.Scan(&userId, &password, &role)
	if err = bcrypt.CompareHashAndPassword(password, []byte(req.Password)); err != nil {
		log.Println("Password is incorrect")
		WriteMessage(response, "Wrong user credentials", 401)
		return
	}
	accessToken := utils.CreateToken(utils.AuthenticatedUser{UserId: userId, Role: role})
	refreshToken := utils.CreateRefreshToken()
	expDays, _ := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXP_DAYS"))
	refreshTokenExp := time.Now().AddDate(0, 0, expDays)
	_, err = restApi.Db.Exec(`INSERT INTO refresh_token(content, expires_at, user_id) VALUES ($1, $2, $3);`, refreshToken, refreshTokenExp, userId)
	fatal(err)
	c := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  refreshTokenExp,
		Path:     "/api/auth/",
		HttpOnly: true,
	}
	http.SetCookie(response, c)
	response.WriteHeader(200)
	data, _ = json.Marshal(struct {
		AccessToken string `json:"accessToken"`
	}{AccessToken: accessToken})
	response.Write(data)
}

func (restApi *RestApi) HandleLogoutRequest(response http.ResponseWriter, request *http.Request) {
	c, err := request.Cookie("refresh_token")
	if err == nil {
		_, err = restApi.Db.Exec("DELETE FROM refresh_token WHERE content=$1", c.Value)
		fatal(err)
		dc := &http.Cookie{
			Name:     "refresh_cookie",
			Value:    "",
			HttpOnly: true,
			Path:     "/api/auth",
		}
		http.SetCookie(response, dc)
	}
	response.WriteHeader(200)
}

func (restApi *RestApi) HandleRefreshTokenRequest(response http.ResponseWriter, request *http.Request) {
	refreshToken, err := request.Cookie("refresh_token")
	if err != nil {
		response.WriteHeader(401)
		return
	}
	log.Println("Tokens refreshing request for ", refreshToken.Value)
	ctx := context.Background()
	tx, err := restApi.Db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	fatal(err)
	rows, err := tx.QueryContext(context.Background(), "SELECT token_id, expires_at, user_id FROM refresh_token WHERE content=$1;", refreshToken.Value)
	if err != nil {
		log.Println("Can't select token")
		tx.Rollback()
	}
	fatal(err)
	if !rows.Next() {
		WriteMessage(response, "Need to login", 401)
		tx.Rollback()
		return
	}
	var (
		tokenId   int
		expiresAt time.Time
		userId    string
	)
	rows.Scan(&tokenId, &expiresAt, &userId)
	rows.Close()
	log.Println("User id: ", userId)
	if time.Now().Compare(expiresAt) >= 0 {
		WriteMessage(response, "Need to login", 401)
		tx.Rollback()
	}
	rows, err = tx.QueryContext(context.Background(), "SELECT name FROM registered_user JOIN roles ON role=role_id WHERE user_id=$1", userId)
	if err != nil {
		log.Println("Can't select user role")
		tx.Rollback()
	}
	fatal(err)
	if rows.Next() {
		var role string
		rows.Scan(&role)
		rows.Close()
		authUser := utils.AuthenticatedUser{
			UserId: userId,
			Role:   role,
		}
		newAccessToken := utils.CreateToken(authUser)
		newRefreshToken := utils.CreateRefreshToken()
		_, err = tx.ExecContext(ctx, "DELETE FROM refresh_token WHERE token_id=$1", tokenId)
		if err != nil {
			tx.Rollback()
		}
		fatal(err)
		expDays, _ := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXP_DAYS"))
		refreshTokenExp := time.Now().AddDate(0, 0, expDays)
		_, err = tx.ExecContext(ctx, "INSERT INTO refresh_token(content, expires_at, user_id) VALUES ($1, $2, $3)", newRefreshToken, refreshTokenExp, userId)
		if err != nil {
			tx.Rollback()
		}
		fatal(err)
		tx.Commit()
		c := &http.Cookie{
			Value:    newRefreshToken,
			HttpOnly: true,
			Path:     "/api/auth/",
			Expires:  refreshTokenExp,
			Name:     "refresh_token",
		}
		http.SetCookie(response, c)
		token := struct {
			AccessToken string `json:"accessToken"`
		}{AccessToken: newAccessToken}
		data, _ := json.Marshal(token)
		response.WriteHeader(200)
		response.Write(data)
		log.Println("Tokens updated successfully, new token: ", newRefreshToken)
	} else {
		WriteMessage(response, "User was deleted", 400)
	}
}
