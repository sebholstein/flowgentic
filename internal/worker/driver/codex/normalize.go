package codex

import "time"

const agent = "codex"

// currentTime returns the current time. It's a variable so tests can override it.
var currentTime = time.Now
