package main

type Article struct {
	Title string `json:"title" bson:"title"`
	Data  string `json:"data" bson:"data"`
}
