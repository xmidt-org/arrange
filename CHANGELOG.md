# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- struct field traversal
- fx.In detection
- support for injecting all options into servers and clients

## [v0.1.5]
- arrangehttp decorators (e.g. middleware) may now be injected

## [v0.1.4]
- http.Roundtripper decoration to support things like metrics and logging

## [v0.1.3]
- Added http.Client support to arrangehttp
- Simpler TLS configuration
- Some better examples in godoc

## [v0.1.2]
- Upgrade mapstructure to v1.3.2
- Added convenient viper.DecoderConfigOption implementations
- Removed UnmarshalExact, as it is superfluous
- Added a simple way to unmarshal multiple keys at once
- Streamlined unmarshal/provide API
- Added arrangehttp, with support for producing http.Server objects from external configuration

## v0.1.0
- Initial creation

[Unreleased]: https://github.com/xmidt-org/arrange/compare/v0.1.5..HEAD
[v0.1.5]: https://github.com/xmidt-org/arrange/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/xmidt-org/arrange/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/xmidt-org/arrange/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/xmidt-org/arrange/compare/v0.1.0...v0.1.2
