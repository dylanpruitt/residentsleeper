# residentsleeper
residentsleeper is a TUI API client created as a clone of Insomnia, so I can get experience developing Go/[Bubbletea](https://github.com/charmbracelet/bubbletea/tree/main). It also comes with a [mock server](https://github.com/dylanpruitt/residentsleeper/blob/main/server/main.go) you can use to test it.


<img width="833" height="552" alt="image" src="https://github.com/user-attachments/assets/f496b500-38a1-4227-ac91-ea1012e609a7" />  

It's intended only for learning purposes, so I don't plan to add a ton to it, but I'm planning to add a few more things to it:
- [ ] ability to swap between POST/GET requests (currently only supports GET)
- [ ] editing request body (you can edit the text area now, but it doesn't actually update the request body)
- [ ] more handlers to mock server for demo purposes

## running residentsleeper
After cloning the repo, you can run residentsleeper using `go run main.go` from the main repo folder. You can additionally run `go run server/main.go` in a separate tab to run the bundled mock server.
