# Changelog

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
