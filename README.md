# Custom Blockchain

```
./node -serve::8080 -newuser:node1.key -newchain:chain1.db -loadaddr:addr.json
./node -serve::9090 -newuser:node2.key -newchain:chain2.db -loadaddr:addr.json
./client -loaduser:node1.key -loadaddr:addr.json
```

Commands:

```
/user balance
/chain tx aaa 3
/chain tx bbb 2
```
