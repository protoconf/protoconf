# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.1.2-rc3.0](https://github.com/protoconf/protoconf/compare/v0.1.2-rc2.0...v0.1.2-rc3.0) (2021-03-25)


### Bug Fixes

* **exec:** allow set protoconf agent addr ([#66](https://github.com/protoconf/protoconf/issues/66)) ([49a662c](https://github.com/protoconf/protoconf/commit/49a662cfe9d299b39d91670f48f602ba08559596))

### [0.1.2-rc2.0](https://github.com/protoconf/protoconf/compare/v0.1.2-rc1.0...v0.1.2-rc2.0) (2020-11-01)


### Features

* **terraform_importer:** add "provider" configuration ([2e341c2](https://github.com/protoconf/protoconf/commit/2e341c2493004b4871a28e7c0571c1c1460bbf1c))


### Bug Fixes

* marshal (u)int64 values as numberic json fields ([8304bf9](https://github.com/protoconf/protoconf/commit/8304bf99bb93751280c597a8f85831ab2c8e2aec))
* release version should start with v ([60e8a19](https://github.com/protoconf/protoconf/commit/60e8a19e4b98feb1316fa205673853a736353d59))

### [0.1.2-rc1.0](https://github.com/protoconf/protoconf/compare/v0.1.2...v0.1.2-rc1.0) (2020-09-20)

### [0.1.2](https://github.com/protoconf/protoconf/compare/v0.1.1...v0.1.2) (2020-08-10)


### Features

* **compiler:** support nested ptypes.Any ([4563ad3](https://github.com/protoconf/protoconf/commit/4563ad36260ee3916c7e823e789b5b8dcec163f3))


### Bug Fixes

* **compiler:** field return NoneType if value is nil ([0317def](https://github.com/protoconf/protoconf/commit/0317def1e589c7c29e62bc89bacbb3d362ef5bd2))
* **import/golang:** set tag before struct visit ([51d33bd](https://github.com/protoconf/protoconf/commit/51d33bdda406115cdb16f2aab8b1d1fa03e265b1))
* **libkv/consul:** fix ([229d9df](https://github.com/protoconf/protoconf/commit/229d9df0c87d663cb0a88e796945ce5190b5e60d))
* **libkv/consul:** revert bad fix ([8c15c72](https://github.com/protoconf/protoconf/commit/8c15c7270d47cd93fea128fb29932cb3e131b7ee))
* **libkv/consul:** tight loop when timeout expires ([b5da114](https://github.com/protoconf/protoconf/commit/b5da114c51b921c68d116211842da1f9d292fcb5))
* **terraform_importer:** support for nested blocks ([c9082ac](https://github.com/protoconf/protoconf/commit/c9082ac3234495aa0eba560862e7b990257b623c))
* **tf:** invalid field names ([1401569](https://github.com/protoconf/protoconf/commit/14015693b2b42edf13aa538fc635f2e1b97df17b))
* image path in docs/architechture ([8a0b3db](https://github.com/protoconf/protoconf/commit/8a0b3db309f07756f8d91a0cb843a5313fcfe2f7))

### [0.1.1-0](https://github.com/protoconf/protoconf/compare/v0.1.1...v0.1.1-0) (2020-08-10)


### Features

* **compiler:** support nested ptypes.Any ([4563ad3](https://github.com/protoconf/protoconf/commit/4563ad36260ee3916c7e823e789b5b8dcec163f3))


### Bug Fixes

* **compiler:** field return NoneType if value is nil ([0317def](https://github.com/protoconf/protoconf/commit/0317def1e589c7c29e62bc89bacbb3d362ef5bd2))
* **import/golang:** set tag before struct visit ([51d33bd](https://github.com/protoconf/protoconf/commit/51d33bdda406115cdb16f2aab8b1d1fa03e265b1))
* **libkv/consul:** fix ([229d9df](https://github.com/protoconf/protoconf/commit/229d9df0c87d663cb0a88e796945ce5190b5e60d))
* **libkv/consul:** revert bad fix ([8c15c72](https://github.com/protoconf/protoconf/commit/8c15c7270d47cd93fea128fb29932cb3e131b7ee))
* **libkv/consul:** tight loop when timeout expires ([b5da114](https://github.com/protoconf/protoconf/commit/b5da114c51b921c68d116211842da1f9d292fcb5))
* **terraform_importer:** support for nested blocks ([c9082ac](https://github.com/protoconf/protoconf/commit/c9082ac3234495aa0eba560862e7b990257b623c))
* **tf:** invalid field names ([1401569](https://github.com/protoconf/protoconf/commit/14015693b2b42edf13aa538fc635f2e1b97df17b))
* image path in docs/architechture ([8a0b3db](https://github.com/protoconf/protoconf/commit/8a0b3db309f07756f8d91a0cb843a5313fcfe2f7))

### [0.1.1](https://github.com/protoconf/protoconf/compare/v0.1.1-beta12.0...v0.1.1) (2020-04-20)


### Features

* **mutate:** add mutate cli ([729d822](https://github.com/protoconf/protoconf/commit/729d822fb011002e24daf45702fac93586e5e140))

### [0.1.1](https://github.com/protoconf/protoconf/compare/v0.1.1-beta12.0...v0.1.1) (2020-04-20)


### Features

* **mutate:** add mutate cli ([729d822](https://github.com/protoconf/protoconf/commit/729d822fb011002e24daf45702fac93586e5e140))

### [0.1.1-beta12.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta11.0...v0.1.1-beta12.0) (2020-04-16)


### Bug Fixes

* **exec:** break loop when channel closed ([bbf4413](https://github.com/protoconf/protoconf/commit/bbf44132de149be1a557b1e719ca7508e8cbd631))

### [0.1.1-beta11.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta10.0...v0.1.1-beta11.0) (2020-04-16)


### Bug Fixes

* **exec:** better loop in exec.go ([82e9969](https://github.com/protoconf/protoconf/commit/82e9969c9fb50762fdf7d9941f86e3141958e18a))

### [0.1.1-beta10.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta9.0...v0.1.1-beta10.0) (2020-04-14)

### [0.1.1-beta9.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta8.0...v0.1.1-beta9.0) (2020-04-13)


### Bug Fixes

* **exec:** move exec setup outside anon func ([906668d](https://github.com/protoconf/protoconf/commit/906668da0830ac81869dd2a0284b5caa89d6ea9c))

### [0.1.1-beta8.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta7.0...v0.1.1-beta8.0) (2020-04-13)


### Bug Fixes

* **exec:** cancel watchers on exec configs updates ([bac10c2](https://github.com/protoconf/protoconf/commit/bac10c2bed40f3952d069ec8d21c56b9b61d5d9d))

### [0.1.1-beta7.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta6.0...v0.1.1-beta7.0) (2020-04-13)


### Bug Fixes

* **importers:** skip visited types when filtering ([669b957](https://github.com/protoconf/protoconf/commit/669b95797a39d58c3d366294334648152e1740e9))
* **importers/golang:** when field is a pointer to underlying struct ([672c6da](https://github.com/protoconf/protoconf/commit/672c6daf901d5f30b4e6b9f2cc236e244360cca9))

### [0.1.1-beta6.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta5.0...v0.1.1-beta6.0) (2020-04-06)


### Features

* **exec:** general purpose protoconf exec ([c87607e](https://github.com/protoconf/protoconf/commit/c87607e4990b1270393fb80d57199c4cc5a8f749))

### [0.1.1-beta5.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta4.0...v0.1.1-beta5.0) (2020-03-26)


### Features

* **compiler:** add support for starlib modules ([2295f7d](https://github.com/protoconf/protoconf/commit/2295f7d340a9a28878ec8ba7bd774f81d5bf3cd6))
* **golang_importer:** import proto files from golang structs ([6c2767e](https://github.com/protoconf/protoconf/commit/6c2767e83d729dde0d4f6c645ca4cbd6d2dfaf09))
* **importers:** add command for golang importer ([679d967](https://github.com/protoconf/protoconf/commit/679d9675773bf5f3e15be63f9ac60d437b964ca4))
* **importers:** filter helpers ([89101cd](https://github.com/protoconf/protoconf/commit/89101cd165fcb47778928b08ee20dc77b53e9062))
* framework for import cfg structs as protobuf ([d4c87e8](https://github.com/protoconf/protoconf/commit/d4c87e80f7eac25bb0e3b051a7d69916689a9e33))
* **pc4tf:** support datasources ([2b49d2d](https://github.com/protoconf/protoconf/commit/2b49d2d00302b3a6567d654da2f23d69fc48c07e))


### Bug Fixes

* **importers:** LoadAllSyntax loading go packages ([42b823e](https://github.com/protoconf/protoconf/commit/42b823e47117f38bd210b4c6d07a08344ce7fa56))

### [0.1.1-beta4.0](https://github.com/protoconf/protoconf/compare/v0.1.1-beta3.0...v0.1.1-beta4.0) (2020-03-19)


### Bug Fixes

* **pc4tf:** nested messages ([d169061](https://github.com/protoconf/protoconf/commit/d16906115106cea66ee80bbaec93bca453f98adc))
* **pc4tf:** replace logrus with zap logger ([c55e8ea](https://github.com/protoconf/protoconf/commit/c55e8eabbe6f3eb75d5af4ac1871bf5b5d686797))

### [0.1.1-beta3.0](https://github.com/protoconf/protoconf/compare/v0.1.1-pre2.0...v0.1.1-beta3.0) (2020-03-18)


### Bug Fixes

* **pc4tf:** object field types to nested messages ([c64368e](https://github.com/protoconf/protoconf/commit/c64368ebbbac5b283a5a643d8931ddff52139def))

### [0.1.1-pre2.0](https://github.com/protoconf/protoconf/compare/v0.1.1-pre1.0...v0.1.1-pre2.0) (2020-02-12)

### 0.1.1-pre1.0 (2020-02-12)


### Features

* **pc4tf:** add meta fields to all resources ([1d8aedd](https://github.com/protoconf/protoconf/commit/1d8aeddbb59a06a763e52b0432bcab3f2694c11f))


### Bug Fixes

* **pc4tf:** set json_name in meta fields ([87dfade](https://github.com/protoconf/protoconf/commit/87dfadeccdd41cf5f02b9196a9b0bca231680c0a))
* **pc4tf:** sort proto files, messages and fields ([5e43908](https://github.com/protoconf/protoconf/commit/5e4390896cccb04a1fb4384d20848e436425ab77))
* **pc4tf:** sort proto message fields ([4a94527](https://github.com/protoconf/protoconf/commit/4a9452788f9bc1aaaf3ec23e0547536eacfa9cd4))

### [0.1.1-0](https://github.com/protoconf/protoconf/compare/v0.1.0-pre2.0...v0.1.1-0) (2020-02-08)


### Features

* **pc4tf:** add meta fields to all resources ([1d8aedd](https://github.com/protoconf/protoconf/commit/1d8aeddbb59a06a763e52b0432bcab3f2694c11f))

### [0.1.1-0](https://github.com/protoconf/protoconf/compare/v0.1.0-pre2.0...v0.1.1-0) (2020-02-08)


### Features

* **pc4tf:** add meta fields to all resources ([1d8aedd](https://github.com/protoconf/protoconf/commit/1d8aeddbb59a06a763e52b0432bcab3f2694c11f))

## [0.1.0-pre2.0](https://github.com/protoconf/protoconf/compare/v0.1.0-pre2...v0.1.0-pre2.0) (2020-01-22)


### Bug Fixes

* **pc4tf:** sort proto message fields ([4a94527](https://github.com/protoconf/protoconf/commit/4a9452788f9bc1aaaf3ec23e0547536eacfa9cd4))
