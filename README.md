# Mins

a mini restful server

to build a efficient restful server with only a command

## usage

mac user

```
wget https://github.com/chenhg5/mins/releases/download/0.0.1/mins_mac -O mins
mins -c /the/config/file/path
```

linux user

```
wget https://github.com/chenhg5/mins/releases/download/0.0.1/mins_linux -O mins
mins -c /the/config/file/path
```

## config.ini example

```
[server]
port = 4006

[database]
addr = localhost
port = 3306
user = root
password = root
database = example
```

## route

| Method     | Path      |
| :-------:  | :-----:   |
| GET        | /resource/:table/id/:id      |
| DELETE     | /resource/:table/id/:id      |
| PUT        | /resource/:table/id/:id      |
| POST       | /resource/:table             |