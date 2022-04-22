Clojure IDE for the Acme text editor.

Intercepts button 2 in any `*.clj` file and sends the selection to a command called `gonrepl`.

Make sure you pass the suitable env vars so it can talk with your nREPL instance.

## Installation

### Dependency

You need to install https://github.com/mkmik/gonrepl

### Binary release

Fetch a binary from https://github.com/mkmik/acmeclj/releases/latest

### Source install 

```bash
go install github.com/mkmik/acmeclj@latest
```

## Usage

The idea is that you edit some code in Acme, select some text with button 2 and send it to a live clojure instance offering a remote REPL session via nREPL.

You need to follow the instructions in the [gonrepl README](https://github.com/mkmik/gonrepl). In particular make sure you have the `LEIN_REPL_PORT` variable correctly set when you run `acmeclj`.

The default nREPL port should usually work. 

```console
$ export LEIN_REPL_PORT=54344
```

If you have this env var when you start the Acme editor you can spawn `acmeclj` from within the editor itself.


## Demo

![acmeclj](https://user-images.githubusercontent.com/52673/164683706-cce07755-aa5d-4e36-bd4a-123a310caed6.gif)
