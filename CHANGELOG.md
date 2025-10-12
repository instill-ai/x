# Changelog

## [0.10.1-alpha](https://github.com/instill-ai/x/compare/v0.10.0-alpha...v0.10.1-alpha) (2025-10-12)


### Bug Fixes

* **client/grpc:** add metadata filtering and service identification ([#67](https://github.com/instill-ai/x/issues/67)) ([baf5c6d](https://github.com/instill-ai/x/commit/baf5c6d0bfbd53eaef9af392ef3ac2f9bc5ee939))

## [0.10.0-alpha](https://github.com/instill-ai/x/compare/v0.9.0-alpha...v0.10.0-alpha) (2025-09-03)


### Features

* **grpc:** add unregistered client constructor ([#64](https://github.com/instill-ai/x/issues/64)) ([a9ac516](https://github.com/instill-ai/x/commit/a9ac516d6f2302b6459502260b78996a8484b33f))
* **temporal:** add support for connecting to Temporal Cloud using API key ([#62](https://github.com/instill-ai/x/issues/62)) ([85de2fb](https://github.com/instill-ai/x/commit/85de2fb234ccbd5ef8e1b797caaed105303f0642))

## [0.9.0-alpha](https://github.com/instill-ai/x/compare/v0.8.0-alpha...v0.9.0-alpha) (2025-07-15)


### Features

* **grpcclient:** add Mgmt and Pipeline gRPC clients ([#51](https://github.com/instill-ai/x/issues/51)) ([d3b2816](https://github.com/instill-ai/x/commit/d3b28166753ab5f1858d43ef0912055a3bc49097))
* **main:** add shared packages ([#53](https://github.com/instill-ai/x/issues/53)) ([035d7f8](https://github.com/instill-ai/x/commit/035d7f80b280af4d9c1482cf03c21ba8041617ed))
* **temporal:** add context propagator ([#47](https://github.com/instill-ai/x/issues/47)) ([9172edd](https://github.com/instill-ai/x/commit/9172edd336bb29ce69be0be124658160b80f58e2))


### Bug Fixes

* **errors:** capture unique key violation constraint Postgres error ([#57](https://github.com/instill-ai/x/issues/57)) ([e330663](https://github.com/instill-ai/x/commit/e330663db2fcc5536a80cd58fc0bf13f90814185))
* **errors:** correct unique key & capture GORM errors ([#58](https://github.com/instill-ai/x/issues/58)) ([b1675d0](https://github.com/instill-ai/x/commit/b1675d0e17954180f7be77a6a1dafc27e82d497f))
* **server:** fix the decider interceptor logic and regex pattern ([#60](https://github.com/instill-ai/x/issues/60)) ([8a7d5d1](https://github.com/instill-ai/x/commit/8a7d5d1ae335f1f18c42cdbd30b3c195743d4a32))


### Miscellaneous

* **client:** improve client package ([#55](https://github.com/instill-ai/x/issues/55)) ([f2d7cce](https://github.com/instill-ai/x/commit/f2d7cce973ee7e3efae60839fb64884193994907))
* **deps:** bump golang.org/x/net from 0.36.0 to 0.38.0 ([#46](https://github.com/instill-ai/x/issues/46)) ([5f87dc0](https://github.com/instill-ai/x/commit/5f87dc044a82d3e2dadd703f44affa012c2f34c1))
* **errors:** improve errors package ([#56](https://github.com/instill-ai/x/issues/56)) ([3829fd8](https://github.com/instill-ai/x/commit/3829fd844cd5c4926e54f9da104c7d4a085bd182))
* **main:** pass lint and remove unused pacakges and files ([#52](https://github.com/instill-ai/x/issues/52)) ([aa51f36](https://github.com/instill-ai/x/commit/aa51f36fb8cfde0ac8e1bef3a8d21634f69acae9))
* **release-please:** update config.json ([#50](https://github.com/instill-ai/x/issues/50)) ([edf022a](https://github.com/instill-ai/x/commit/edf022aae3333f750ffe88f7fd27d1d46783ed5f))
* **server:** improve server package ([#54](https://github.com/instill-ai/x/issues/54)) ([ad47957](https://github.com/instill-ai/x/commit/ad479570e6331d32a1b89548189ee2204b3ee559))


### Refactor

* **client,server:** improve gRPC server and client options ([#61](https://github.com/instill-ai/x/issues/61)) ([6376b6c](https://github.com/instill-ai/x/commit/6376b6cc6d5e4282b5d0f6466567c6fa3c9edf2c))
* **minimock,quicktest:** unify testing framework ([#59](https://github.com/instill-ai/x/issues/59)) ([87222c2](https://github.com/instill-ai/x/commit/87222c286bdbb132a769f8fdbf2eaf83af0a6879))

## [0.8.0-alpha](https://github.com/instill-ai/x/compare/v0.7.0-alpha...v0.8.0-alpha) (2025-03-25)


### Features

* **minio:** add private blob upload ([#43](https://github.com/instill-ai/x/issues/43)) ([bd27e04](https://github.com/instill-ai/x/commit/bd27e04e1246009e9b7dbe7008e768add5450929))
* **temporal:** add metrics to temporal client ([#45](https://github.com/instill-ai/x/issues/45)) ([dbca7ae](https://github.com/instill-ai/x/commit/dbca7ae1a0f2f1280bd85c65c738615cbaa56ffc))

## [0.7.0-alpha](https://github.com/instill-ai/x/compare/v0.6.0-alpha...v0.7.0-alpha) (2025-02-24)


### Features

* **minio:** add app info to MinIO client ([#37](https://github.com/instill-ai/x/issues/37)) ([8000506](https://github.com/instill-ai/x/commit/8000506aa4551228dd1f52cfca225fab206d9afd))
* **minio:** add client info and user header to bucketless client ([#39](https://github.com/instill-ai/x/issues/39)) ([87c3450](https://github.com/instill-ai/x/commit/87c34501e6cdc86546a900c1f3269ec5fb5ce2b9))
* **minio:** expose SDK client to ease migration to x/minio ([#40](https://github.com/instill-ai/x/issues/40)) ([be48bc7](https://github.com/instill-ai/x/commit/be48bc78368d145e643aa1270b38f57636ecf586))
* **minio:** log MinIO actions with requester ([#34](https://github.com/instill-ai/x/issues/34)) ([1b1559c](https://github.com/instill-ai/x/commit/1b1559c35b51e698a53472bae61e925354f816a0))
* **minio:** pass UserUID as header and delegate logging to MinIO ([#36](https://github.com/instill-ai/x/issues/36)) ([6af31ff](https://github.com/instill-ai/x/commit/6af31ff7cc27ca2f14c00fa5c932798a7a0d09f1))
* **minio:** test minio package on CI ([#35](https://github.com/instill-ai/x/issues/35)) ([e3ab78c](https://github.com/instill-ai/x/commit/e3ab78c6d1b81ae494fd9b1d2819207f4ea59b98))


### Bug Fixes

* **mod:** update golang.org/x/net module to fix vulnerability issue ([#32](https://github.com/instill-ai/x/issues/32)) ([0c9fa95](https://github.com/instill-ai/x/commit/0c9fa957ecaa076dda39e901ac887c3a31d08f99))

## [0.6.0-alpha](https://github.com/instill-ai/x/compare/v0.5.0-alpha...v0.6.0-alpha) (2024-12-13)


### Features

* collect shared logic for blob storage and minio ([#27](https://github.com/instill-ai/x/issues/27)) ([36280f1](https://github.com/instill-ai/x/commit/36280f1781206f99f176732964d6ce9080d2f288))
* **minio:** add expiry tag and rule ([#23](https://github.com/instill-ai/x/issues/23)) ([6659d46](https://github.com/instill-ai/x/commit/6659d4662da56fd7af36034b3756f856607d61de))


### Bug Fixes

* **minio:** remove default bucket lifecycle rule ([#30](https://github.com/instill-ai/x/issues/30)) ([890bb31](https://github.com/instill-ai/x/commit/890bb310fcb2f236b798044212850cdaf4fb63d3))
* **minio:** set life cycle config on existing bucket ([#25](https://github.com/instill-ai/x/issues/25)) ([3b853d0](https://github.com/instill-ai/x/commit/3b853d0b8656d116798e31cffa2db4dab84724a2))

## [0.5.0-alpha](https://github.com/instill-ai/x/compare/v0.4.0-alpha...v0.5.0-alpha) (2024-10-03)


### Features

* **errmsg:** support errors wrapped through errors.Join ([#21](https://github.com/instill-ai/x/issues/21)) ([5ced228](https://github.com/instill-ai/x/commit/5ced228b749839129417cdd5214daad774ce043d))

## [0.4.0-alpha](https://github.com/instill-ai/x/compare/v0.3.0-alpha...v0.4.0-alpha) (2023-12-19)


### Features

* add package to handle end-user error messages. ([#13](https://github.com/instill-ai/x/issues/13)) ([6230a89](https://github.com/instill-ai/x/commit/6230a89e386c9135fcadcaddb76ffa052fba82ea))

## [0.3.0-alpha](https://github.com/instill-ai/x/compare/v0.2.0-alpha...v0.3.0-alpha) (2023-04-23)


### Features

* add temporal client option wrappers ([4e89cdb](https://github.com/instill-ai/x/commit/4e89cdb95a96ff44f2fb02c01b296a30ca1f87f7))

## [0.2.0-alpha](https://github.com/instill-ai/x/compare/v0.1.0-alpha...v0.2.0-alpha) (2022-07-06)


### Features

* add google rpc status error details ([bceeac6](https://github.com/instill-ai/x/commit/bceeac65f5232dc15c9176ea39c10e4bda3cb238))
* **checkfield:** add protobuf annotation check package ([e81f88d](https://github.com/instill-ai/x/commit/e81f88dda39bd7cb26355a7706abc4696840d441))
* **paginate:** add package ([5a70916](https://github.com/instill-ai/x/commit/5a70916ce4258602d069262476be23478e8e44c5))
* **repo:** add package ([39fcffc](https://github.com/instill-ai/x/commit/39fcffc82edb43cf739040deea94b5e67c8cacb6))
* **structutil:** add protobuf struct util package ([d98c6e1](https://github.com/instill-ai/x/commit/d98c6e13153fc3b6e09d1785ee0d792bd3cd8d01))


### Bug Fixes

* **checkfield:** fix checkfield path recursion ([30c5644](https://github.com/instill-ai/x/commit/30c56444b8f3556b88cf6c014dc501c1b68da758))
* **checkfield:** fix empty message reflect.ValueOf issue ([a0cc7c9](https://github.com/instill-ai/x/commit/a0cc7c979c669803cc08ebbb82c2bd7b19f91d69))
* **checkfield:** fix immutable pointer struct logic ([baab8aa](https://github.com/instill-ai/x/commit/baab8aaa93b22745e3e1a1a64cb7a4fb120c4b6c))
* **checkfield:** fix struct immutable check ([9afb850](https://github.com/instill-ai/x/commit/9afb85044c1c4d86acea5a521108ceb8f46d2cc2))
* refactor checkfield functions and add tests ([#3](https://github.com/instill-ai/x/issues/3)) ([83b1d7b](https://github.com/instill-ai/x/commit/83b1d7b1bffd04b39bb007affc3c5beb1ade6ae0))

## [0.1.0-alpha](https://github.com/instill-ai/x/compare/v0.0.0-alpha...v0.1.0-alpha) (2022-02-25)


### Features

* add zapadapter ([0606624](https://github.com/instill-ai/x/commit/06066245ff82ba2c03441c0810a3ba7316bc7514))
