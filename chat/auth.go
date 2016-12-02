package main
import (
    "net/http"
    "strings"
    "fmt"
    "log"
)
type authHandler struct {
    next http.Handler
}
func (h *authHandler) ServeHTTP(w http.ResponseWriter,
                                r *http.Request) {
    if _, err := r.Cookie("auth"); err == http.ErrNoCookie {
        // not authenticated
        w.Header().Set("Location", "/login")
        w.WriteHeader(http.StatusTemporaryRedirect)
    } else if err != nil {
        // some other error
        panic(err.Error())
    } else {
        // success - call the next handler
        h.next.ServeHTTP(w, r)
    }
}
func MustAuth(handler http.Handler) http.Handler {
    return &authHandler{next: handler}
}
// loginHandler handles the third-party login process.
// format: /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
    segs := strings.Split(r.URL.Path, "/")
    log.Println("Length of segs: ", len(segs), "segs: ", segs)
    if len(segs) < 4 {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintf(w, "Invalid auth location %s", r.URL.Path)
    } else {
        action := segs[2]
        provider := segs[3]
        switch action {
        case "login":
            log.Println("TODO handle login for" , provider)
        default:
            w.WriteHeader(http.StatusNotFound)
            fmt.Fprintf(w, "Auth action %s not supported", action)
        }
    }
}
