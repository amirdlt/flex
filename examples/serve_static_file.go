package main

import . "github.com/amirdlt/flex"

func staticFileServe() {
	s := Default()

	s.GET("/api-doc", func(i *BasicInjector) Result {
		fileContent := []byte(`<!DOCTYPE html>
<html>
<head>
    <title>Hello, World!</title>
</head>
<body>
    <h1>Hello, World!</h1>
</body>
</html>
`)
		i.SetContentType("text/html")
		return i.WrapOk(fileContent)
	})

	_ = s.Run(":2048")
}
