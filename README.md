# arrange

Arrange is a companion to go.uber.org/fx that adds unmarshaled components, conditional options, and some other goodies.  Refer to the godoc for more information and examples.

[![Build Status](https://github.com/xmidt-org/arrange/workflows/CI/badge.svg)](https://github.com/xmidt-org/arrange/actions)
[![codecov.io](http://codecov.io/github/xmidt-org/arrange/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/arrange?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/arrange)](https://goreportcard.com/report/github.com/xmidt-org/arrange)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/arrange/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/arrange.svg)](CHANGELOG.md)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xmidt-org/arrange)](https://pkg.go.dev/github.com/xmidt-org/arrange)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_arrange&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_arrange)

## Summary

Arrange provides an integration with [uber/fx](https://pkg.go.dev/go.uber.org/fx?tab=doc) and the following libraries:

- [viper](https://pkg.go.dev/github.com/spf13/viper?tab=doc) is used for unmarshaling and driving the state of components from external configuration
- [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux?tab=doc) is supplied for all unmarshaled servers as the root handler.  Dependency injection code can customize a mux.Router for each server, typically inside an fx.Invoke function.
- [zap](https://pkg.go.dev/go.uber.org/zap?tab=doc) is supported as a logging infrastructure.  Arrange does not directly refer to zap, but it supply adapters that conform to zap's API pattern.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Details](#details)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/).
By participating, you agree to this Code.

## Install

go get github.com/xmidt-org/arrange

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
