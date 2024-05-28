package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

type AuthUserInfo string

type AuthenticatedUser struct {
	UserId string
	Role   string
}

type UserClaims struct {
	*jwt.StandardClaims
	Role string `json:"role"`
}

func ValidateToken(tokenString string) (*AuthenticatedUser, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("receive token with wrong encryption method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return nil, fmt.Errorf("can't parse token: %v", err)
	}
	userClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("can't parse token: invalid token claims")
	}
	return &AuthenticatedUser{
		UserId: userClaims["sub"].(string),
		Role:   userClaims["role"].(string),
	}, nil
}

func CreateToken(authUser AuthenticatedUser) string {
	expDur, err := time.ParseDuration(os.Getenv("ACCESS_TOKEN_EXP_SECONDS") + "s")
	if err != nil {
		log.Fatal(err)
	}
	expAt := time.Now().Add(expDur).Unix()
	claims := &UserClaims{
		&jwt.StandardClaims{
			Subject:   authUser.UserId,
			Issuer:    os.Getenv("ISSUER"),
			ExpiresAt: expAt,
			IssuedAt:  time.Now().Unix(),
		},
		authUser.Role,
	}
	token := jwt.New(jwt.SigningMethodHS512)
	token.Claims = claims
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		log.Fatal(err)
	}
	return tokenString
}

func CreateRefreshToken() string {
	dict := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := make([]byte, 100)
	for i := 0; i < 100; i++ {
		bigInt, _ := rand.Int(rand.Reader, big.NewInt(62))
		s[i] = dict[bigInt.Uint64()]
	}
	return string(s)
}

func ExtractAuthUserFromRequest(request *http.Request) (*AuthenticatedUser, error) {
	user, ok := request.Context().Value(AuthUserInfo("user")).(*AuthenticatedUser)
	if !ok {
		return nil, errors.New("can't get auth user from context")
	}
	return user, nil
}
