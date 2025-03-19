## Instructions to run

```
go run cmd/main.go
```

## Instructions to stop

Currently need to kill the terminal to stop the app. Ctrl + C or Cmd + C doesn't work.

## SQL Support

This app currently supports only 3 commands:
1. CREATE
    Syntax:
    ```
    CREATE TABLE <table-name> (<column-name> <column-type>, ...);
    ```

2. INSERT
    Syntax:
    ```
    INSERT INTO <table-name> VALUES (<value>);
    ```

    Note: Supports one insert at a time.

3. SELECT
    Syntax:
    ```
    SELECT <column-name>, ... FROM <table-name>;
    ```

    Note: Doesn't support Select * statements


## Supported Data Types

1. INT for integers
2. TEXT for strings (should be in single quotes)