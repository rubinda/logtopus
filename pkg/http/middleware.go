package http

import "net/http"

// authMiddleware ensures a valid token is present before handing the request over to the next handler.
func authMiddleware(jwtAuth *JWTAuthority, endpointHandler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Token"] == nil {
			jsonResponse(w, http.StatusUnauthorized, errResponse{ErrTokenMissing.Error(), nil})
			return
		}
		if jwtAuth == nil {
			jsonResponse(w, http.StatusInternalServerError, errResponse{"Can't authenticate your request, please contact an administrator.", nil})
			return
		}
		_, err := jwtAuth.ValidateToken(r.Header["Token"][0])
		if err != nil {
			jsonResponse(w, http.StatusUnauthorized, errResponse{err.Error(), nil})
			return
		}

		// All ok, continue with the handler stack
		endpointHandler(w, r)
	})
}
