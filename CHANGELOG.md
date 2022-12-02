# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic
Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## [0.4.0] - 2022-12-02

### Changed
- Update Sensu Go and SDK dependencies with the correct modules
- Support PEM certificate

## [0.3.0] - 2020-03-19
- The handler now checks for the "deactivated" attribute after requesting node
information, and uses its presence to determine if the node does not exist.

## [0.2.0] - 2020-02-11

### Changed
- The Puppet node name can now be overridden using entities annotations

## [0.1.0] - 2020-02-11

### Added
- Initial release
