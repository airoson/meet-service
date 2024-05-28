package controllers

import (
	"context"
	"log"
	"meet-service/utils"
	"net/http"
)

func GetContentTypeMiddleware(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Content-Type", contentType)
			next.ServeHTTP(response, request)
		})
	}
}

func checkExistence(val string, in []string) bool {
	for _, v := range in {
		if v == val {
			return true
		}
	}
	return false
}

func GetJWTValidation(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if checkExistence("UNAUTHORIZED", allowedRoles) {
				next.ServeHTTP(response, request)
				return
			}
			tokenString := request.Header.Get("Authorization")
			if len(tokenString) < 7 || tokenString[:7] != "Bearer " {
				log.Printf("Can't authorize request to %s: invalid authorization header\n", request.URL.String())
				response.WriteHeader(401)
				return
			}
			authUser, err := utils.ValidateToken(tokenString[7:])
			if err != nil {
				log.Printf("Can't authorize request to %s: user not authenticated: %v\n", request.URL.String(), err)
				response.WriteHeader(401)
				return
			}

			if !checkExistence(authUser.Role, allowedRoles) {
				log.Printf("Can't authorize request to %s: user doesn't have permission to access resource\n", request.URL.String())
				response.WriteHeader(401)
				return
			}
			ctx := context.WithValue(request.Context(), utils.AuthUserInfo("user"), authUser)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func HandlePanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				response.WriteHeader(400)
				log.Printf("Error in handler: %v\n", err)
			}
		}()
		next.ServeHTTP(response, request)
	})
}

// func GetAllowOrigins(origins ...string) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
// 			origin := strings.Join(origins, ",")
// 			log.Printf("Ready to add headers, response is nil: %v\n", response == nil)
// 			response.Header().Set("Access-Control-Allow-Origin", origin)
// 			response.Header().Set("Access-Control-Allow-Headers", "*")
// 			log.Println("Complete, request is nil: ", request == nil)
// 			if request.Method == "OPTION" {
// 				return
// 			}
// 			log.Println("next is nil: ", next == nil)
// 			if next != nil {
// 				next.ServeHTTP(response, request)
// 			}
// 		})
// 	}
// }
