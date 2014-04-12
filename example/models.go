package main

import (
	"fmt"
	"io"
	"time"
)

// A Greet is a 140-character micro-blogpost that has no resemblance whatsoever
// to the noise a bird makes.
type Greet struct {
	User    string    `param:"user"`
	Message string    `param:"message"`
	Time    time.Time `param:"time"`
}

// Store all our greets in a big list in memory, because, let's be honest, who's
// actually going to use a service that only allows you to post 140-character
// messages?
var Greets = []Greet{
	{"carl", "Welcome to Gritter!", time.Now()},
	{"alice", "Wanna know a secret?", time.Now()},
	{"bob", "Okay!", time.Now()},
	{"eve", "I'm listening...", time.Now()},
}

// Write out a representation of the greet
func (g Greet) Write(w io.Writer) {
	fmt.Fprintf(w, "%s\n@%s at %s\n---\n", g.Message, g.User,
		g.Time.Format(time.UnixDate))
}

// A User is a person. It may even be someone you know. Or a rabbit. Hard to say
// from here.
type User struct {
	Name, Bio string
}

// All the users we know about! There aren't very many...
var Users = map[string]User{
	"alice": {"Alice in Wonderland", "Eating mushrooms"},
	"bob":   {"Bob the Builder", "Making children dumber"},
	"carl":  {"Carl Jackson", "Duct tape aficionado"},
}

// Write out the user
func (u User) Write(w io.Writer, handle string) {
	fmt.Fprintf(w, "%s (@%s)\n%s\n", u.Name, handle, u.Bio)
}
