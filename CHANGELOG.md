## [1.1.23](https://github.com/hardpointlabs/agent/compare/v1.1.22...v1.1.23) (2026-04-10)


### Bug Fixes

* align paths in go config ([8042a71](https://github.com/hardpointlabs/agent/commit/8042a71e65cb65f89b3fad6397b9a2811a4db9ef))

## [1.1.22](https://github.com/hardpointlabs/agent/compare/v1.1.21...v1.1.22) (2026-04-10)


### Bug Fixes

* remove /var/log/hardpoint from ReadWritePaths ([92d2c28](https://github.com/hardpointlabs/agent/commit/92d2c28ced77eb725af248bbdbd0267456b425bb))

## [1.1.21](https://github.com/hardpointlabs/agent/compare/v1.1.20...v1.1.21) (2026-04-10)


### Bug Fixes

* postinst typo ([603c2cd](https://github.com/hardpointlabs/agent/commit/603c2cd795ced7ca0ebe3d057a0dbba67694ef46))

## [1.1.20](https://github.com/hardpointlabs/agent/compare/v1.1.19...v1.1.20) (2026-04-10)


### Bug Fixes

* add systemd unit ([264bf95](https://github.com/hardpointlabs/agent/commit/264bf958920f8589d267b97fa163e2a45ca08055))

## [1.1.19](https://github.com/hardpointlabs/agent/compare/v1.1.18...v1.1.19) (2026-04-09)


### Bug Fixes

* missing GPG_PASSPHRASE for goreleaser ([6ffed3b](https://github.com/hardpointlabs/agent/commit/6ffed3b26c674175e5dbe9bb053cc451f2618a9a))

## [1.1.18](https://github.com/hardpointlabs/agent/compare/v1.1.17...v1.1.18) (2026-04-09)


### Bug Fixes

* remove sudo, use native aptly config for R2, remove GPG_KEY_PATH ([316695f](https://github.com/hardpointlabs/agent/commit/316695f4dcc8ce3c65a3a712f7f2955d2b93bcdb))

## [1.1.17](https://github.com/hardpointlabs/agent/compare/v1.1.16...v1.1.17) (2026-04-09)


### Bug Fixes

* pinentry mode on import ([0f1afee](https://github.com/hardpointlabs/agent/commit/0f1afee2ef4561494cc159d53c3f4a2b9efde939))

## [1.1.16](https://github.com/hardpointlabs/agent/compare/v1.1.15...v1.1.16) (2026-04-09)


### Bug Fixes

* feed passphrase in from tempfile ([839ccea](https://github.com/hardpointlabs/agent/commit/839ccea2e1a302bf9a51d8de7f60d1319ab359bb))

## [1.1.15](https://github.com/hardpointlabs/agent/compare/v1.1.14...v1.1.15) (2026-04-09)


### Bug Fixes

* nFPM signer syntax ([2bcc52d](https://github.com/hardpointlabs/agent/commit/2bcc52d84a57403c8c0a21d09fa5aaaccf6d7918))

## [1.1.14](https://github.com/hardpointlabs/agent/compare/v1.1.13...v1.1.14) (2026-04-09)


### Bug Fixes

* no base64 ([ee944e8](https://github.com/hardpointlabs/agent/commit/ee944e856522bc3e312fb6b2cc01ed8380c98296))

## [1.1.13](https://github.com/hardpointlabs/agent/compare/v1.1.12...v1.1.13) (2026-04-09)


### Bug Fixes

* pgp signing ([f80f11e](https://github.com/hardpointlabs/agent/commit/f80f11e4b35ddfac35869f6926a7ff215afda9f5))

## [1.1.12](https://github.com/hardpointlabs/agent/compare/v1.1.11...v1.1.12) (2026-04-09)


### Bug Fixes

* install aptly through apt ([9da9a90](https://github.com/hardpointlabs/agent/commit/9da9a900f5421e8f04a346bfd7b7f9306e29e8bc))

## [1.1.11](https://github.com/hardpointlabs/agent/compare/v1.1.10...v1.1.11) (2026-04-09)


### Bug Fixes

* publish .debs to R2 ([7a1e5aa](https://github.com/hardpointlabs/agent/commit/7a1e5aa4334cd80ebfaf7fd30874f49a250f0445))

## [1.1.10](https://github.com/hardpointlabs/agent/compare/v1.1.9...v1.1.10) (2026-04-08)


### Bug Fixes

* avoid logging agent start before arg parsing ([a906cd8](https://github.com/hardpointlabs/agent/commit/a906cd811628bcc20598ae2ae473f2cbf3809ddb))

## [1.1.9](https://github.com/hardpointlabs/agent/compare/v1.1.8...v1.1.9) (2026-04-08)


### Bug Fixes

* annotations, not labels ([d3b011c](https://github.com/hardpointlabs/agent/commit/d3b011c93e597d32a52831125296057c0552ea8a))

## [1.1.8](https://github.com/hardpointlabs/agent/compare/v1.1.7...v1.1.8) (2026-04-08)


### Bug Fixes

* add OCI source label ([c065bc1](https://github.com/hardpointlabs/agent/commit/c065bc1558c7437ac53435479d9b6922129ef80f))

## [1.1.7](https://github.com/hardpointlabs/agent/compare/v1.1.6...v1.1.7) (2026-04-08)


### Bug Fixes

* set bare:true when publishing images ([766ce9b](https://github.com/hardpointlabs/agent/commit/766ce9b0e67fc16e71d4e68f1286cd835b6598b3))

## [1.1.6](https://github.com/hardpointlabs/agent/compare/v1.1.5...v1.1.6) (2026-04-08)


### Bug Fixes

* install syft before generating sbom ([859e6b1](https://github.com/hardpointlabs/agent/commit/859e6b169c641e43036971e1decbf34254bbba65))

## [1.1.5](https://github.com/hardpointlabs/agent/compare/v1.1.4...v1.1.5) (2026-04-08)


### Bug Fixes

* build docker images thru ko, release on GHCR ([095782c](https://github.com/hardpointlabs/agent/commit/095782ca305bc299b4e5516c6709f5d26a5cfa45))

## [1.1.4](https://github.com/hardpointlabs/agent/compare/v1.1.3...v1.1.4) (2026-04-07)


### Bug Fixes

* boilerplate debian support ([ac608ff](https://github.com/hardpointlabs/agent/commit/ac608ff7b327c9cfd3bac2d74d49e5c19f68d7d6))

## [1.1.3](https://github.com/hardpointlabs/agent/compare/v1.1.2...v1.1.3) (2026-04-07)


### Bug Fixes

* bump checkout and semantic-release actions ([5e50f82](https://github.com/hardpointlabs/agent/commit/5e50f82e639bbf355dc17fcfd1e46f3cf6c9c38b))

## [1.1.2](https://github.com/hardpointlabs/agent/compare/v1.1.1...v1.1.2) (2026-04-07)


### Bug Fixes

* use default directory value ([974a934](https://github.com/hardpointlabs/agent/commit/974a93410598fa3f02fea106c5ec5fcbae8fb398))

## [1.1.1](https://github.com/hardpointlabs/agent/compare/v1.1.0...v1.1.1) (2026-04-07)


### Bug Fixes

* update goreleaser config to >v2.26 ([72fd76b](https://github.com/hardpointlabs/agent/commit/72fd76b6b3967754af486da4b451f3fb694320cf))

# [1.1.0](https://github.com/hardpointlabs/agent/compare/v1.0.0...v1.1.0) (2026-04-07)


### Features

* release with official goreleaser action ([d2c5d73](https://github.com/hardpointlabs/agent/commit/d2c5d73bb6e3f4d81a805d7d4eb018e6774a30b7))

# 1.0.0 (2026-04-07)


### Features

* initial release ([e00f231](https://github.com/hardpointlabs/agent/commit/e00f231ef71f1b8a5cf80b65acf67ae875830538))
