# residentsleeper
residentsleeper is a TUI API client created as a clone of Insomnia, so I can get experience developing Go/[Bubbletea](https://github.com/charmbracelet/bubbletea/tree/main). It also comes with a [mock server](https://github.com/dylanpruitt/residentsleeper/blob/main/server/main.go) you can use to test it.

![20251219-2128-30 7450918](https://github.com/user-attachments/assets/cbc7cac5-0be2-43c5-a990-c75f5ffa22a2)

## running residentsleeper
After cloning the repo, you can run residentsleeper using `go run main.go` from the main repo folder. You can additionally run `go run server/main.go` in a separate tab to run the bundled mock server.

You'll probably need to resize the window to be bigger - I've been fixing a few funky things with UI in the default terminal window size, but parts of the UI might get cut off otherwise. Unfortunately, I don't know of any way to set the terminal width/height in bubbletea or I'd do that.  
I've also provided a few queries you can use with the mock server to demo the client's functionality.

When creating a header or query, it should be in a `name:value` format. Anything that doesn't follow that format will not be saved after editing it.
