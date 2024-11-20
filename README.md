# Rock Paper Scissors Website with a very janky matchmaking system that totally doesn't have ANY security vulnerabilities or anything else that'll come back to haunt me

The site is hosted locally on my machine, using cloudflare for tunneling, DNS, as well as TLS as needed by the .dev extenstion.

https://toopsi.dev

## Build instructions

**1. Install go**

Install instructions for go:
https://go.dev/dl/

**2. Install templ**

Install instructions for templ:
https://templ.guide/quick-start/installation

**3. Transpile .templ files to go code**

From the project root directory run: 
```
templ generate
```

**4. Compile and run the program**

To just build run:
```
go build ./pkg
```

or

To build and run at the same time run:
```
go run ./pkg`
```
