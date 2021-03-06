package main

import (
    "log"
    "net/http"
    "text/template"
    "path/filepath"
    "sync"
    "flag"
    "errors"
    "fmt"
    "io/ioutil"
    "github.com/stretchr/gomniauth"
    "github.com/stretchr/gomniauth/providers/github"
    "github.com/stretchr/gomniauth/providers/facebook"
    "github.com/stretchr/gomniauth/providers/google"
    "github.com/stretchr/objx"
    "gopkg.in/yaml.v2"
)
// set the active Avatar implementation
var avatars Avatar = TryAvatars{
    UseFileSystemAvatar,
    UseAuthAvatar,
    UseGravatar,
}
type instanceConfig struct {
    SecurityKey  string `yaml:"security_key"`
    Google       map[string]string
    Facebook     map[string]string
    GitHub       map[string]string
}
func (c *instanceConfig) Parse(data []byte) error {
    if err := yaml.Unmarshal(data, c); err != nil {
        return err
    }
    if c.SecurityKey == "" {
        return errors.New("Chat config: invalid security key")
    }
    if _, ok := c.Google["clientid"]; !ok {
        fmt.Println("Google['clientid']: ", c.Google["clientid"])
        return errors.New("Chat config: invalid Google oauth id")
    }
    if _, ok := c.Google["clientsecret"]; !ok {
        return errors.New("Chat config: invalid Google oauth secret")
    }
    if _, ok := c.Facebook["clientid"]; !ok {
        return errors.New("Chat config: invalid Facebook oauth id")
    }
    if _, ok := c.Facebook["clientsecret"]; !ok {
        return errors.New("Chat config: invalid Facebook oauth secret")
    }
    if _, ok := c.GitHub["clientid"]; !ok {
        return errors.New("Chat config: invalid GitHub oauth id")
    }
    if _, ok := c.GitHub["clientsecret"]; !ok {
        return errors.New("Chat config: invalid GitHub oauth secret")
    }
    return nil
}

// templ represents a single template
type templateHandler struct {
    once      sync.Once
    filename  string
    templ     *template.Template
}

// ServeHTTP handles the HTTP request.
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    t.once.Do(func() {
        t.templ = template.
                  Must(template.
                       ParseFiles(filepath.
                                  Join("templates",
                                       t.filename)))
    })
    data := map[string]interface{}{"Host": r.Host,}
    if authCookie, err := r.Cookie("auth"); err == nil {
        data["UserData"] = objx.MustFromBase64(authCookie.Value)
    }
    t.templ.Execute(w, data)
}

func main() {
    var addr = flag.String("addr", ":8080", "The addr of the application.")
    flag.Parse() // parse the flags
    // set up gomniauth
    data, err := ioutil.ReadFile("secrets.yaml")
    if err != nil {
        log.Fatal(err)
    }
    var configs instanceConfig
    if err := configs.Parse(data); err != nil {
        log.Fatal(err)
    }
    callbackStub := "http://localhost%s/auth/callback/%s"
    gomniauth.SetSecurityKey(configs.SecurityKey)
    gomniauth.WithProviders(
        facebook.New(configs.Facebook["clientid"],
                     configs.Facebook["clientsecret"],
                     fmt.Sprintf(callbackStub, *addr, "facebook"),
                    ),
        github.New(configs.GitHub["clientid"],
                     configs.GitHub["clientsecret"],
                     fmt.Sprintf(callbackStub, *addr, "github"),
                  ),
        google.New(configs.Google["clientid"],
                     configs.Google["clientsecret"],
                     fmt.Sprintf(callbackStub, *addr, "google"),
                  ),
    )

    r := newRoom()
    http.Handle("/", MustAuth(&templateHandler{filename: "login.html"}))
    http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
    http.Handle("/login", &templateHandler{filename: "login.html"})
    http.Handle("/upload", &templateHandler{filename: "upload.html"})
    http.Handle("/avatars/",
        http.StripPrefix("/avatars/",
            http.FileServer(http.Dir("./avatars"))))
    http.HandleFunc("/auth/", loginHandler)
    http.HandleFunc("/uploader", uploaderHandler)
    http.HandleFunc("/logout/", func(w http.ResponseWriter,
                                     r *http.Request) {
      http.SetCookie(w, &http.Cookie{
          Name:   "auth",
          Value:  "",
          Path:   "/",
          MaxAge: -1,
      })
      w.Header()["Location"] = []string{"/chat"}
      w.WriteHeader(http.StatusTemporaryRedirect)
    })
    http.Handle("/room", r)
    // get the room going
    go r.run()
    // start the web server
    log.Println("Starting web server on", *addr)
    if err:= http.ListenAndServe(*addr, nil); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
