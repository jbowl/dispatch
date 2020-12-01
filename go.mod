module github.com/jbowl/dispatch

go 1.15

require (
	github.com/gorilla/mux v1.8.0
	github.com/jbowl/apibrewery v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
	google.golang.org/grpc v1.33.2

)

//replace github.com/jbowl/apibrewery => /home/j/jsoft/github.com/jbowl/apibrewery

//replace github.com/jbowl/apibrewery => /home/j/jsoft/github.com/jbowl/dispatch/apibrewery
replace github.com/jbowl/apibrewery => ./apibrewery
