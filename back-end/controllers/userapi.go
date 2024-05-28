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
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
}

func (restApi *RestApi) HandleUserGetRequest(response http.ResponseWriter, request *http.Request) {
	user, err := utils.ExtractAuthUserFromRequest(request)
	if err != nil {
		log.Printf("Error: %v\n", err)
		response.WriteHeader(400)
		return
	}
	userId := user.UserId
	reqIds := request.URL.Query()["userId"]
	if reqIds != nil {
		reqId := reqIds[0]
		if user.Role != "ADMIN" && reqId != userId {
			response.WriteHeader(401)
			return
		}
		userId = reqId
	}
	rows, err := restApi.Db.Query("SELECT phone, email, shown_name FROM registered_user WHERE user_id = $1", userId)
	fatal(err)
	if rows.Next() {
		var (
			phone sql.NullString
			email sql.NullString
			name  sql.NullString
		)
		rows.Scan(&phone, &email, &name)
		info := UserInfo{
			UserId: userId,
			Phone:  phone.String,
			Name:   name.String,
			Email:  email.String,
		}
		data, _ := json.Marshal(info)
		response.WriteHeader(200)
		response.Write(data)
	} else {
		WriteMessage(response, "This user was deleted", 400)
	}
}

func (restApi *RestApi) HandleUserPatchRequest(response http.ResponseWriter, request *http.Request) {
	user, err := utils.ExtractAuthUserFromRequest(request)
	if err != nil {
		log.Printf("Error: %v\n", err)
		response.WriteHeader(400)
		return
	}
	req := UserInfo{}
	data, _ := io.ReadAll(request.Body)
	err = json.Unmarshal(data, &req)
	if err != nil {
		WriteMessage(response, "Wrong user data", 400)
		return
	}
	userId := user.UserId
	if req.UserId != "" && userId != req.UserId {
		if user.Role == "Admin" {
			userId = req.UserId
		} else {
			response.WriteHeader(401)
			return
		}
	}
	params := []interface{}{}
	clauseVars := []string{}
	if req.Email != "" {
		if !emailReg.Match([]byte(req.Email)) {
			WriteMessage(response, "Invalid email", 400)
			return
		}
		params = append(params, req.Email)
		clauseVars = append(clauseVars, fmt.Sprintf("email=$%d", len(params)))
	}
	if req.Phone != "" {
		if !phoneReg.Match([]byte(req.Phone)) {
			WriteMessage(response, "Invalid phone", 400)
			return
		}
		params = append(params, req.Phone)
		clauseVars = append(clauseVars, fmt.Sprintf("phone=$%d", len(params)))
	}
	ctx := context.Background()
	tx, err := restApi.Db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	fatal(err)
	if len(params) != 0 {
		params = append(params, userId)
		whereClause := "SELECT user_id FROM registered_user WHERE (" + strings.Join(clauseVars, " OR ") + fmt.Sprintf(") AND user_id != $%d", len(params))
		rows, err := tx.QueryContext(ctx, whereClause, params...)
		if err != nil {
			tx.Rollback()
		}
		fatal(err)
		if rows.Next() {
			errMsg := clauseVars[0]
			if len(clauseVars) > 1 {
				errMsg += " or " + clauseVars[1]
			}
			WriteMessage(response, errMsg, 400)
			tx.Rollback()
			return
		}
		params = params[:len(params)-1]
	}
	if req.Name != "" {
		params = append(params, req.Name)
		clauseVars = append(clauseVars, fmt.Sprintf("shown_name=$%d", len(params)))
	}
	if req.Password != "" {
		encryptedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		fatal(err)
		params = append(params, encryptedPass)
		clauseVars = append(clauseVars, fmt.Sprintf("password=$%d", len(params)))
	}
	if len(params) != 0 {
		params = append(params, userId)
		setClause := "UPDATE registered_user SET " + strings.Join(clauseVars, ", ") + fmt.Sprintf(" WHERE user_id=$%d;", len(params))
		_, err = tx.ExecContext(ctx, setClause, params...)
		if err != nil {
			tx.Rollback()
		}
		fatal(err)
	}
	rows, err := tx.QueryContext(ctx, "SELECT email, phone, shown_name FROM registered_user WHERE user_id = $1", userId)
	if err != nil {
		tx.Rollback()
	}
	fatal(err)
	if rows.Next() {
		var (
			email sql.NullString
			phone sql.NullString
			name  sql.NullString
		)
		err = rows.Scan(&email, &phone, &name)
		if err != nil {
			log.Printf("Can't select updated user: %v", err)
		}
		err = tx.Commit()
		if err != nil {
			response.WriteHeader(400)
			return
		}
		updated := UserInfo{
			UserId: userId,
			Phone:  phone.String,
			Email:  email.String,
			Name:   name.String,
		}
		data, _ := json.Marshal(updated)
		response.WriteHeader(200)
		response.Write(data)
	} else {
		err = tx.Commit()
		if err != nil {
			response.WriteHeader(400)
			return
		}
		WriteMessage(response, fmt.Sprintf("Can't find user with id %s", userId), 404)
	}
}
