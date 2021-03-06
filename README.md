# clang-format-gae

GAE based web-wrapper around clang-format

## About

**clang-format-gae** wraps *clang-format* in a simple HTTP server, making it
possible to format source files with a POST request. It is meant to run on
*Google AppEngine* (GAE) using a *custom runtime environment*. This is required
in order to be able to call *clang-format*.

## Usage

Once running, code files can be formatted with anything that can do a POST
request. For example:

```bash
curl --data-binary '@-' https://format.collectd.org <input.c >output.c
```

The script `contrib/format.sh` can be used to format files in-place.

## License

[ISC License](https://opensource.org/licenses/ISC)

## Author

Florian Forster
