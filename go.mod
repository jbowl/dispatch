module github.com/jbowl/dispatch

go 1.15

require (
	github.com/gorilla/mux v1.8.0
	github.com/jbowl/apibrewery v0.0.0-20201201014425-0aabf5982cbd
	github.com/sirupsen/logrus v1.7.0
	google.golang.org/grpc v1.33.2
)

//replace github.com/jbowl/apibrewery => ./apibrewery