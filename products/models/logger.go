package models

import "time"

type Logger struct {
	Method     string    `bson:"method" json:"method"`
	Url        string    `bson:"url" json:"url"`
	UserAgent  string    `bson:"user_agent" json:"user_agent"`
	Body       string    `bson:"body" json:"body"`
	Ip         string    `bson:"ip" json:"ip"`
	Email      string    `bson:"email" json:"email"`
	StatusCode int       `bson:"status_code" json:"status_code"`
	Response   string    `bson:"response" json:"response"`
	CreateAt   time.Time `bson:"created_at" json:"created_at"`
}
