# purlinfo

[![codecov](https://codecov.io/gh/boringbin/purlinfo/graph/badge.svg?token=I8O0SZC11X)](https://codecov.io/gh/boringbin/purlinfo)

A simple CLI tool to get information about a package from a [purl](https://github.com/package-url/purl-spec).

Uses the [Ecosyste.ms](https://ecosyste.ms/) API to get information about a package.

## Usage

```text
Usage: purlinfo [OPTIONS] purl

Get package information from a package URL (purl).

Arguments:
  purl    Package URL (e.g., pkg:npm/lodash@4.17.21)

Options:
  -json
        Output as JSON
  -timeout duration
        HTTP request timeout (default 30s)
  -v    Verbose output (debug mode)
  -version
        Show version and exit
```

## License

[MIT](LICENSE)
