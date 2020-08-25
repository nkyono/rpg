# rpg
Reddit post getter. Fills and queries local postgresql database.

Use:
    (Can also now just set values in .env file)
    USERNAME=... PASSWORD=... AGENT=... PUBLIC=... PRIVATE=... go run reddit.go [-red]
    where USER, PASS is the reddit username and password. AGENT is the app user-agent. PUBLIC, PRIVATE are the public and secret keys provided by reddit. Optional flag -red is flag to tell program to request from reddit (defualt false); mainly made for testing purposes.

