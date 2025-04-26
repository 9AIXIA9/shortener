
<a name="v1.1.0"></a>
## [v1.1.0](https://github.com/9AIXIA9/shortener/compare/v1.0.0...v1.1.0) (2025-04-26)

### Feat

* add connection configuration and enhance URL checking logic with retries
* implement sequence caching with Redis and local storage, enhancing ID generation efficiency
* add todo.md to .gitignore for better task management

### Fix

* improve error handling in local cache ID filling and streamline URL path configuration
* the issue of connectivity checking and format of URL
* the issue that the next path of the same domain name can be connected to all of them in the connectivity test

### Pull Requests

* Merge pull request [#1](https://github.com/9AIXIA9/shortener/issues/1) from 9AIXIA9/refactor/optimization-response


<a name="v1.0.0"></a>
## v1.0.0 (2025-04-21)

### Feat

* update configuration structure and enhance sequence generation logic
* restructure configuration types for improved clarity and organization
* enhance error handling and refactor URL processing logic
* add handlers and logic for URL shortening and retrieval

### Fix

* fixed the link verification logic bug and optimized the validate interface

### Refactor

* rename ConvertLogic to ShortenLogic and update related types and tests
* update .gitignore with additional rules for logs, Go binaries, and environment files
* update .gitignore with additional rules for logs, Go binaries, and environment files
* add unit tests for ConvertLogic with various scenarios
* update import paths and add Domain configuration to AppConf
* change log level from Info to Debug for improved debugging information
* add sensitive word detection and update configuration for sensitive words
* implement base62 encoding and decoding with tests
* implement client interface and enhance URL handling in conversion logic
* implement sequence and short URL map repositories with database interactions
* update dependencies and clean up URL parsing logic
* enhance URL validation and error handling in conversion logic

