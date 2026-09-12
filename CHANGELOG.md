# Changelog

## [2.0.0](https://github.com/rknightion/grafana-cloud-vending-machine/compare/v1.0.1...v2.0.0) (2026-09-12)


### ⚠ BREAKING CHANGES

* GrafanaProvisioningRepository requests written against 1.x are refused at admission until they are restructured. The required url, branch and path fields moved off spec.repository into a block named by a new required type discriminator, which selects one of six provider shapes; sync and workflows are now required rather than supplied by the renderer; stackRef.name and repository.uid became immutable, so an update that changes either is refused; and the create form of every secure.<key> field is unrepresentable. No other request kind changes. See docs/migration-1.0.md#migrating-to-20.

### Features

* vend Git Sync provisioning end to end ([cbfdb73](https://github.com/rknightion/grafana-cloud-vending-machine/commit/cbfdb737a81dcc61dc6acf5c79f703f107f0810d))


### Bug Fixes

* quote optional test filters containing shell metacharacters ([b0a3b37](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b0a3b37a8513c49f8b9a89436d5735be8594ad30))
* vend a Git Sync connection Grafana Cloud will accept ([e65313f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e65313fd684aeb52d88bdfcae72d6aebc74dd1c9))


### Documentation

* **backlog:** a shared prerequisite parks its consumers, never the whole run ([a015fee](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a015fee68e1022fe7962a26cd8394540a580af16))
* **backlog:** GCV-0070 and GCV-0071 - the Git Sync surface is one third vendable ([b43192c](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b43192c58d89a4d62c957aca2233a0da85f57eb8))
* **backlog:** GCV-0074 - the vended Git Sync connection cannot be created ([e6a7666](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e6a76660f9f2ba79ed68b2c72c11577de50aba75))
* record the 1.x to 2.0 GrafanaProvisioningRepository migration ([87844be](https://github.com/rknightion/grafana-cloud-vending-machine/commit/87844be3ac5a5cfffa74522f8e12f792e0be8034))


### Miscellaneous Chores

* close wave 11 delivery tasks ([3d789c8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3d789c8b8530fdd405155753ce1a05580d736817))
* park wave 10 on fixture provenance mismatch ([e624f16](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e624f16fb85aab10a35fdb13bbf154cf4511e3f9))
* pin published vending function ([2be1af8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/2be1af8ae9f6bb48639c6baeeb604b29518ed921))
* pin published vending function ([46abc2f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/46abc2f816d9cf3bf6c1c8f90f9fca4e5be0a538))
* record provider kind mismatch ([9c5f1b7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/9c5f1b731e91b7e5ca560fb988500a8bc7951628))

## [1.0.1](https://github.com/rknightion/grafana-cloud-vending-machine/compare/v1.0.0...v1.0.1) (2026-09-11)


### Bug Fixes

* take the publication scan's customer identifiers from the environment ([f2c15ab](https://github.com/rknightion/grafana-cloud-vending-machine/commit/f2c15ab3a6d7f4fb627eda464294443723e3ef6c))


### Documentation

* align recovery contracts and compaction evidence ([9137442](https://github.com/rknightion/grafana-cloud-vending-machine/commit/9137442cf2dfa8275dff34d4c1c498b14b00ad58))
* **backlog:** a vended stack's claim never reaches Ready without an in-stack resource ([4c88c67](https://github.com/rknightion/grafana-cloud-vending-machine/commit/4c88c67e3c736cbe63dd2b3840f0abbb59dff2fd))
* **backlog:** GCV-0069 - the alerting and oncall APIs cannot express an IRM chat destination ([446b3c4](https://github.com/rknightion/grafana-cloud-vending-machine/commit/446b3c43495cbe2c07f5a8f96ea79212719fe5c4))
* sync root async question policy ([c0685db](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c0685db75848b618c5c78ad5c4e926dc25d5c30f))
* sync same-session fan-out recovery contract ([43cdec0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/43cdec078489c8c0b21234adfec7297a6dfd099a))

## [1.0.0](https://github.com/rknightion/grafana-cloud-vending-machine/compare/v0.1.0...v1.0.0) (2026-09-09)


### ⚠ BREAKING CHANGES

* add governed vending surfaces and bounded token lifetimes
* add multi-organization vending and domain modules

### Features

* add consumable comprehensive reference ([3b3e4b0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3b3e4b06b77c1efb0e3569f9d347f1f7f57bd18a)), closes [#3](https://github.com/rknightion/grafana-cloud-vending-machine/issues/3)
* add governed vending surfaces and bounded token lifetimes ([2368b38](https://github.com/rknightion/grafana-cloud-vending-machine/commit/2368b38e256ef45718bf4f5eff55f5920604f381))
* add multi-organization vending and domain modules ([1c48c7e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1c48c7e376a2671182457baeb084a30ebe0a56aa))
* add multi-organization vending seams ([e79211c](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e79211c2fbdeeb4f93b55ffbf35d93b93ef69394))
* add portable Grafana Cloud vending reference ([c5b0453](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c5b0453549db8d829c95ef3dce52aef2d9549c8e))
* **ci:** add a ci-success aggregator ([f92b2da](https://github.com/rknightion/grafana-cloud-vending-machine/commit/f92b2dadd364d0e5afa8c886cbd0d23b00e27441))
* expand SSO and access reference ([7c3a37b](https://github.com/rknightion/grafana-cloud-vending-machine/commit/7c3a37bdd4825ec326b70e0ebf90d85a78d38584))
* freeze fail-closed wave 3 vending seams ([e25ede8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e25ede82066eee7173b545882c471c109820df88))
* **function:** stage stack-local resources behind the stack they need ([e2a660f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e2a660f8a6be4abda53d07a110b6d4321001113e))
* harden stack lifecycle and access staging ([659861f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/659861f8ed01a199d8167608067b78527a7a19cb))
* provision the pinned envtest assets locally ([3380150](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3380150d8db42e717049f891f234c0930dcf7e4d))
* support authorized cross-cluster stack consumption ([15b4df5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/15b4df59f51391b1f492110276211d0b929c4c9d))
* vend alert routing, on-call and bounded cloud product surfaces ([c321631](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c3216315be49d1059c1086e9feb8ca3fa62f9a89))
* vend bounded opt-in Asserts configurations ([0d48499](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0d484998adaf9d30181a27f9606bf042eefe6fe2))


### Bug Fixes

* author is Rob Knight, not Rob Knighton ([9816b7f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/9816b7fba74e1eb1599e25e1d3ae103f8508fe63))
* **ci:** narrow the publication scan to real environment identity ([df56993](https://github.com/rknightion/grafana-cloud-vending-machine/commit/df569934fa280307173b084197bd6d5b413064bb))
* close wave 6 review findings ([d1713e7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/d1713e78d0969ad58569e9d207a6ecf5c8dbedd3))
* correct the documented function digest and gate it ([7cbf10e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/7cbf10e04855f35c1ab069fb6b141de34c6b58fa))
* **deps:** move k8s.io/api and controller-runtime together, and group them ([85c4344](https://github.com/rknightion/grafana-cloud-vending-machine/commit/85c4344a5146eea98b4bfa9fb1c110858cd1f152))
* **deps:** move the function package pin to the signed security build ([8b602cc](https://github.com/rknightion/grafana-cloud-vending-machine/commit/8b602cc092c102493d35f5341288e754d293acdd))
* **deps:** update module github.com/crossplane/crossplane-runtime/v2 to v2.4.0 ([#30](https://github.com/rknightion/grafana-cloud-vending-machine/issues/30)) ([302c90a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/302c90a08ddf050aadcfc9b429dff490a908fdd3))
* **deps:** update module github.com/crossplane/crossplane/apis/v2 to v2.4.0 ([#9](https://github.com/rknightion/grafana-cloud-vending-machine/issues/9)) ([6294c24](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6294c248458d7e7590c5844607c8525ddd967c71))
* harden child adoption contracts ([fcd3bc5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/fcd3bc5b57b178b0d62fd97f93509229506cf318)), closes [#4](https://github.com/rknightion/grafana-cloud-vending-machine/issues/4)
* make minimal catalog consumable ([7f57a56](https://github.com/rknightion/grafana-cloud-vending-machine/commit/7f57a5609d5203ca05ce96520a840cdc99ad3a50)), closes [#3](https://github.com/rknightion/grafana-cloud-vending-machine/issues/3)
* pin wave 7 function package ([efb82fc](https://github.com/rknightion/grafana-cloud-vending-machine/commit/efb82fcf4a209a51d2fc64c9119edb2357909927))
* **platform:** stop pinning the runtime ServiceAccount names ([3e286de](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3e286dee6d49ea94413d3ec1e9447d9c950daebb))
* prevent public documentation drift ([5e6c0c2](https://github.com/rknightion/grafana-cloud-vending-machine/commit/5e6c0c259726c88158af71e0fc8a9e7f0cb31a4c))
* render every catalog directory in the gate, not a hand-kept list ([28b4832](https://github.com/rknightion/grafana-cloud-vending-machine/commit/28b4832de3bfafd5e3c0faa3a2bff31dc4bffbad))
* repair admission preconditions before wave 5 ([4d70189](https://github.com/rknightion/grafana-cloud-vending-machine/commit/4d701896969cb26c3d697a0357d6da701ecaa4cf))
* **scan:** allow the shared CI tooling and runner-pool references ([52654c7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/52654c7bf1180c0492a9a877f5dc6f71a47ab6d8))


### Documentation

* **agents:** grafana-cloud-vending-machine to AGENTS.md standard ([fbe2aa6](https://github.com/rknightion/grafana-cloud-vending-machine/commit/fbe2aa6ca6991f4a5378c446ae9e543f44d972d9))
* **backlog:** a consumer cluster cannot mint against a stack it did not vend ([d9ff94e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/d9ff94e3c742856e53fdd234b80703f9a4d5635c))
* **backlog:** complete Go 1.27 upgrade ([daee800](https://github.com/rknightion/grafana-cloud-vending-machine/commit/daee8005692319a6303eaa5e8fd49bf32cebd056))
* **backlog:** complete justfile migration ([1b16200](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1b162003ec234d111b8f5f59e33d1c12d7978ac9))
* **backlog:** correct the recorded asserts vending coverage to zero ([59b4b14](https://github.com/rknightion/grafana-cloud-vending-machine/commit/59b4b149f7235ac4d1f8c96fbcaea10f5751d9e1))
* **backlog:** open the family-derivation blind spot wave 8 left unnamed ([90094fd](https://github.com/rknightion/grafana-cloud-vending-machine/commit/90094fdc1bfe32666e37d816ecc0bc993a2323e7))
* **backlog:** open the prose inventory drift GCV-0057 did not cover ([4de03fb](https://github.com/rknightion/grafana-cloud-vending-machine/commit/4de03fbabfcb0145d6d0f6695860feaa1554ff96))
* **backlog:** record acceptance evidence for GCV-0064 to GCV-0067 ([ec57913](https://github.com/rknightion/grafana-cloud-vending-machine/commit/ec579134cbb7c96169e2a3adf54bec04b174d959))
* **backlog:** record that the motivating migration did not exercise GCV-0061 ([c8d6cae](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c8d6caee2d0f97343fc53753fec993a5d9bdd5c5))
* **backlog:** sync fan-out protocol — CodeRabbit review gate ([1989590](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1989590aa3566b9a770b476360a6e3ef9de73da1))
* **backlog:** sync fan-out protocol — success criteria vs write authority ([ae7603f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/ae7603f279ed21525286ced2471363e072fe0524))
* close all nine wave 3 tasks with delivery evidence ([6bf671b](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6bf671b4cf9c1540c07096108e0b9c7cea873498))
* close the SCIM lifecycle question and open the coverage queue ([b7be8a0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b7be8a045d33ee64401f939fe9af0676507e0878))
* correct the measured provider pin reference count ([867700b](https://github.com/rknightion/grafana-cloud-vending-machine/commit/867700b6dc0ec64c5c937924c4ac02f0b335c7b9))
* finalize wave 9 acceptance evidence ([81051ca](https://github.com/rknightion/grafana-cloud-vending-machine/commit/81051ca77548449550ffa40add213d172609ef55))
* open GCV-0041 for the hosted admission-gate prerequisite ([f6c5bdb](https://github.com/rknightion/grafana-cloud-vending-machine/commit/f6c5bdbbe2428f011bcc4978d767fc0dc2d02356))
* open the wave 4 queue as GCV-0034 to GCV-0039 ([1de3dd8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1de3dd809dde57c12da431184c0213a28b2b6073))
* open the wave 5 queue and record the granted schema authority ([da53a75](https://github.com/rknightion/grafana-cloud-vending-machine/commit/da53a757376a41e1c39ecf3288d4541a7670dfca))
* point the issue index at the archive now the issues are gone ([0870ce8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0870ce8c2bef1e2d295c751bfe8513885f6516e5))
* publish a documentation site and join the m7kni.io fleet ([a9ab075](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a9ab0756e07b6a68c346f25daff5b36cf2ed699e))
* re-import fan-out protocol (context-cost rules) ([6fe0563](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6fe05634c90788505b35053d651ead6c7bd1f17b))
* re-import the fan-out protocol at c1e6cb0 ([c74e8a7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c74e8a75c2d0fcdc89afe3ec1ea5a988eb25cbda))
* re-render the fan-out protocol from agent-docs ([a937ef0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a937ef01aec9fad589b538c22b632bb1eb63ff63))
* re-render the fan-out protocol from agent-docs 711db6c ([0baa42d](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0baa42de30005ceeabf269a1a40a97df41572f86))
* re-render the fan-out protocol from agent-docs b0d76d8 ([a4bd019](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a4bd019802e4ebea2daeac3636c257738f6c5bfc))
* reconcile wave 4 delivery and admission parks ([4f947d0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/4f947d0c7d94aa8d115c1467922bcb201b93ec6a))
* record admission parks and retire README serialization ([5c482ff](https://github.com/rknightion/grafana-cloud-vending-machine/commit/5c482ffcf2100714cfe1c3751733d805a501489a))
* record the provisioned release-please broker consumer on GCV-0033 ([ee33f65](https://github.com/rknightion/grafana-cloud-vending-machine/commit/ee33f650f8b8ef4a2d6d3ac6d225e4f7630af914))
* record wave 5 admission outcomes ([3c4579a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3c4579a509b5be302dc97560bf1773cfc438b098))
* retire source-held credentials, and settle the live-proof boundary ([12fedd6](https://github.com/rknightion/grafana-cloud-vending-machine/commit/12fedd6b6c33e1128585158922b5ebb499715d71))
* separate enabled requests from examples ([c478b9f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c478b9f57f002aa0c834839d531df8d9bdb07415))
* settle synthetic monitoring boundaries ([b1ceaa3](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b1ceaa36d63dc8b449215b9b362103d7da9b1dbf))
* state Kubernetes admission prerequisites ([b998937](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b998937b1af2ccaf852bc677602b813dcce850d4))
* sync agent-docs, a wave's launch message is a file not a chat block ([69134ef](https://github.com/rknightion/grafana-cloud-vending-machine/commit/69134ef44395cbfcc5d6a5129e21aaaebdff3af8))
* sync Astra routing and default wave reports to files ([2118389](https://github.com/rknightion/grafana-cloud-vending-machine/commit/2118389c3dec2080bec48e06ddadd0d748bfda3b))
* sync authorised Astra root judgement ([e478c5e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e478c5e20ee206ccddaaebe68dc091adcca80c4b))
* sync authorised Astra root judgement ([f4b3bc9](https://github.com/rknightion/grafana-cloud-vending-machine/commit/f4b3bc9b1a375295580f2cd7cb0602071c533f2c))
* sync fan-out protocol from agent-docs ad7abd2 ([cae40d8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/cae40d82f6a22f3459d546d7925034e80b167150))
* sync fan-out protocol, delegated root authority for unattended runs ([34d1299](https://github.com/rknightion/grafana-cloud-vending-machine/commit/34d12990b2173a5e1e79b01b8db07ede90080c94))
* sync fan-out protocol, explicit add does not bound the commit ([9af868e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/9af868e2b574bb13da11fbf81d81407593cb8366))
* sync nineteen-worker Codex fan-out guidance ([a633b63](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a633b63330d446117462e638574a5ac1308d7b3f))
* sync optional Astra fan-out routing and run contracts ([a0851df](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a0851df40d8205ae44fea4b0404be0b2fcefad41))
* sync wave-root stage authority and lab-Mac GUI gate ([819e1c2](https://github.com/rknightion/grafana-cloud-vending-machine/commit/819e1c27c84db396776f46bf532b7534520e2388))
* track multi-org vending, release tooling and the wave-1 conventions ([42f6a0b](https://github.com/rknightion/grafana-cloud-vending-machine/commit/42f6a0bf052ddeeb9384088718039ce6ef5c7dee))
* **tracker:** align canonical fan-out protocol ([16d9fb5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/16d9fb5b9ae0eee14101980f35eb99d1628d1d84))
* **tracker:** close GCV-0002, runtime ServiceAccount names unpinned ([b1656e7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b1656e75d463f9a52ae6ea7c87a051009a3a6518))
* **tracker:** close GCV-0003 through GCV-0005, log GCV-0006 through GCV-0009 ([3909d37](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3909d37a4bfe21324bb31840eceb3fbc55f79a35))
* **tracker:** close GCV-0006 through GCV-0009 ([a461965](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a461965b6a3ae57af5c3998fdef1fe17d6a7b3f1))
* **tracker:** close GCV-0010 with the completing commit and validation run ([60933d7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/60933d76cb3e8c841c332b2a694d5204d0af9f0a))
* **tracker:** correct the canonical owner in the rendered header ([76f44f5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/76f44f5513534a05e29e167444c9e5d8ef804e8a))
* **tracker:** normalise the closed-issues doc title ([6d51580](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6d515805cb69955705d349ae2c22c3daea603766))
* **tracker:** open the API expansion queue as GCV-0010 to GCV-0029 ([b683d02](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b683d02f562491f312605775016b4c61376d1fab))
* **tracker:** re-import the fan-out protocol from canonical ([7bfb09b](https://github.com/rknightion/grafana-cloud-vending-machine/commit/7bfb09b4330e658d2f4968aa4741473b9a1fc4ae))
* **tracker:** record the Adaptive Metrics provider feasibility findings on GCV-0029 ([bfa9d59](https://github.com/rknightion/grafana-cloud-vending-machine/commit/bfa9d59a8aebcf7569bc15a6781e0d14efb3dd9e))
* **tracker:** render agent documents from the canonical source ([e9a9af8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e9a9af821147e31d12ae2f972875d9083ac817ab))


### Miscellaneous Chores

* align CodeRabbit review configuration ([a2637a3](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a2637a3f721401ff955f29a93d46370a7f0ac745))
* **backlog:** add GCV-0031 — migrate the repo task surface to just ([6fe69e5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6fe69e5595b1bdbaef2ac114a3b5b9ef90b92026))
* **backlog:** close the measured explicit-null admission boundary ([b6745e1](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b6745e1f03251d33255aee0965292ce5af6a8b1f))
* **backlog:** close the package reconciliation and correct the SCIM mechanism ([e1acf70](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e1acf700ec55cc445b9dbaa19095457f65c7d185))
* **backlog:** close wave 7 consolidation ([6abab1a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6abab1a7c5d7686f0ef103cad0b60f195d73b4b1))
* **backlog:** open the wave 7 consolidation queue ([36826f8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/36826f8e015a86e762f2d1b6fdee8d8c3f72323f))
* **backlog:** ratify ci as the sanctioned superset of check ([e7d6984](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e7d69846081e6b6b068c9794bba27a9334cbc8a5))
* **backlog:** reconcile wave 6 evidence and precise partial outcomes ([85d1241](https://github.com/rknightion/grafana-cloud-vending-machine/commit/85d1241643e9d06ece85e87ad89e191d23604dd8))
* **backlog:** record package publication boundary ([2a6e3b2](https://github.com/rknightion/grafana-cloud-vending-machine/commit/2a6e3b20c199c02b978a23c4be942bc163eca688))
* **backlog:** track function package reconciliation ([a4d4ecb](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a4d4ecbe79da088cdd70a4800661ac19c1d8a2f5))
* **backlog:** wire the fleet migration ordering into this task ([38f91c8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/38f91c863eec6d746be132757f03355b55545d2c))
* **deps:** update dependency go to v1.27.0 ([#7](https://github.com/rknightion/grafana-cloud-vending-machine/issues/7)) ([6f56932](https://github.com/rknightion/grafana-cloud-vending-machine/commit/6f56932f76d2f60c605ea5b5266ff172fee0ac89))
* **deps:** update dependency go to v1.27.1 ([#25](https://github.com/rknightion/grafana-cloud-vending-machine/issues/25)) ([c770afa](https://github.com/rknightion/grafana-cloud-vending-machine/commit/c770afa8922af33c8300c09d6cea1292006aa714))
* **deps:** update docker/dockerfile:1 docker digest to ecfaec9 ([#5](https://github.com/rknightion/grafana-cloud-vending-machine/issues/5)) ([12fa8aa](https://github.com/rknightion/grafana-cloud-vending-machine/commit/12fa8aa21bb0a1db8472e6d5273e7a0b425013bc))
* **deps:** update extractions/setup-just action to v4 ([#14](https://github.com/rknightion/grafana-cloud-vending-machine/issues/14)) ([0b3f4e1](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0b3f4e1cfc99f21e76ab1c818d8c6d35e9db4081))
* **deps:** update golang docker tag to v1.27.0 ([#8](https://github.com/rknightion/grafana-cloud-vending-machine/issues/8)) ([57ebe81](https://github.com/rknightion/grafana-cloud-vending-machine/commit/57ebe811aa7b5c7dc9885ac142347ba4c0d860a6))
* **deps:** update golang docker tag to v1.27.1 ([#26](https://github.com/rknightion/grafana-cloud-vending-machine/issues/26)) ([8a77573](https://github.com/rknightion/grafana-cloud-vending-machine/commit/8a7757352bdc98f2bdff85cd0252d61a417281a2))
* **deps:** update golang:1.27.0 docker digest to 0ecdc2a ([#12](https://github.com/rknightion/grafana-cloud-vending-machine/issues/12)) ([8c56497](https://github.com/rknightion/grafana-cloud-vending-machine/commit/8c56497717ce0873b6e89e5ae461a91a2530e87e))
* **deps:** update golang:1.27.0 docker digest to 4013ae0 ([#22](https://github.com/rknightion/grafana-cloud-vending-machine/issues/22)) ([31b787a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/31b787a135bbd4b4e9e3b2be6090d23e1092f0f4))
* **deps:** update module github.com/crossplane/crossplane-runtime/v2 to v2.3.3 [security] ([#13](https://github.com/rknightion/grafana-cloud-vending-machine/issues/13)) ([f748184](https://github.com/rknightion/grafana-cloud-vending-machine/commit/f748184f43f8e4ada52e6ab44636eeeb3565a8ed))
* **deps:** update module google.golang.org/grpc to v1.83.1 [security] ([#24](https://github.com/rknightion/grafana-cloud-vending-machine/issues/24)) ([dd8e23d](https://github.com/rknightion/grafana-cloud-vending-machine/commit/dd8e23d44899cf5ff665656d2ffa4c22e782c13c))
* **deps:** update module google.golang.org/grpc to v1.83.2 [security] ([#31](https://github.com/rknightion/grafana-cloud-vending-machine/issues/31)) ([629c39c](https://github.com/rknightion/grafana-cloud-vending-machine/commit/629c39c2b42e6294df0bdc3c59442846940779f0))
* **deps:** update rknightion/.github action to v1.10.0 ([#15](https://github.com/rknightion/grafana-cloud-vending-machine/issues/15)) ([7ec38d0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/7ec38d0d396f55eea025e07b87df702ad5bf2db8))
* **deps:** update rknightion/.github action to v1.11.0 ([#16](https://github.com/rknightion/grafana-cloud-vending-machine/issues/16)) ([14b55eb](https://github.com/rknightion/grafana-cloud-vending-machine/commit/14b55ebe7bb0de1e75b1c43575a65b8e567bda7a))
* **deps:** update rknightion/.github action to v1.13.0 ([#17](https://github.com/rknightion/grafana-cloud-vending-machine/issues/17)) ([db0d0b0](https://github.com/rknightion/grafana-cloud-vending-machine/commit/db0d0b04b5a278c48f14d9b3aec2594b80557075))
* **deps:** update rknightion/.github action to v1.15.0 ([#18](https://github.com/rknightion/grafana-cloud-vending-machine/issues/18)) ([255b6e7](https://github.com/rknightion/grafana-cloud-vending-machine/commit/255b6e79a3bdeb064191b2dd3ca8f58590cf6fec))
* **deps:** update rknightion/.github action to v1.15.1 ([#19](https://github.com/rknightion/grafana-cloud-vending-machine/issues/19)) ([de97c7e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/de97c7eb82879ed8873cd9677d6f8296a612b2fb))
* **deps:** update rknightion/.github action to v1.17.0 ([#20](https://github.com/rknightion/grafana-cloud-vending-machine/issues/20)) ([df87608](https://github.com/rknightion/grafana-cloud-vending-machine/commit/df87608a3b40ff29642e21e43df94c9dffda2493))
* **deps:** update rknightion/.github action to v1.17.1 ([#21](https://github.com/rknightion/grafana-cloud-vending-machine/issues/21)) ([0d19db3](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0d19db3d86bc6a80588b154275f7e8f8338d1885))
* **deps:** update rknightion/.github action to v1.18.0 ([#23](https://github.com/rknightion/grafana-cloud-vending-machine/issues/23)) ([1cfbc94](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1cfbc9432cf53f05171424f013451ff50ae23fc1))
* **deps:** update rknightion/.github action to v1.20.0 ([#27](https://github.com/rknightion/grafana-cloud-vending-machine/issues/27)) ([a09f5b3](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a09f5b31d3a5c4944fb272ac67b3d5eb4ed0a16a))
* **deps:** update rknightion/.github action to v1.21.0 ([#28](https://github.com/rknightion/grafana-cloud-vending-machine/issues/28)) ([e5181a5](https://github.com/rknightion/grafana-cloud-vending-machine/commit/e5181a5ad5604ae686068b49cfb84f0e54c63b8a))
* drop the per-repo Backlog.md guard, now global in the agent config ([0d485ac](https://github.com/rknightion/grafana-cloud-vending-machine/commit/0d485ac0c36dc2828959e2bd3e92f14560d8ce8c))
* finalize wave 2 tasks ([27731ef](https://github.com/rknightion/grafana-cloud-vending-machine/commit/27731ef6ac21378f1ce88316229630599c11136f))
* **function:** pin lifecycle package digest ([d8d8ee9](https://github.com/rknightion/grafana-cloud-vending-machine/commit/d8d8ee9d90d714249178e468f908721e805a6af8))
* **function:** pin the staged-render package digest ([9a2b73d](https://github.com/rknightion/grafana-cloud-vending-machine/commit/9a2b73d031313a556bd5a3fea68127d8b47358fe))
* gitignore the codex agent scratch directory ([a8f6b12](https://github.com/rknightion/grafana-cloud-vending-machine/commit/a8f6b1225155970c13482ddeb381efbfc3edee52))
* migrate work tracking to Backlog.md ([2f4d69a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/2f4d69a0a5bdb6d13386955e8e797ff8754a24e9))
* park wave 1 after route verification stop ([bf90f9a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/bf90f9a4e4adc1c5cd841d785b51ed0c2b227d92))
* pin comprehensive reference function ([3fd7a04](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3fd7a0447f1bb194a0183d7404c996d8568bafe7)), closes [#1](https://github.com/rknightion/grafana-cloud-vending-machine/issues/1)
* pin hardened vending function ([63e2eab](https://github.com/rknightion/grafana-cloud-vending-machine/commit/63e2eabe4494b7bd88dce97f60498f6e82900fcf)), closes [#4](https://github.com/rknightion/grafana-cloud-vending-machine/issues/4)
* pin signed vending function package ([b51fe2f](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b51fe2f8ba39ed3b85273f75fff3c7cb61ed61fe))
* pin signed wave 3 vending function ([bec9551](https://github.com/rknightion/grafana-cloud-vending-machine/commit/bec9551c3c2abb009a4a50412b33efe47b07520c))
* pin the published vending function ([83f81af](https://github.com/rknightion/grafana-cloud-vending-machine/commit/83f81afee7526fd6e7c4ec0a47675774d00036b8))
* pin the verified Asserts function package ([3faf9a8](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3faf9a8b02a92d3777f42753c212e2bed346cf7f))
* pin the verified wave 6 function package ([187b03e](https://github.com/rknightion/grafana-cloud-vending-machine/commit/187b03ea40ee32fcea890e40c138f00a8c73bd5f))
* pin verified cross-cluster consumer function ([3314fb9](https://github.com/rknightion/grafana-cloud-vending-machine/commit/3314fb91bed68f4a4d3fe48b5c469a45a270f85a))
* record wave 8 acceptance evidence ([88b26f3](https://github.com/rknightion/grafana-cloud-vending-machine/commit/88b26f392150d6395294077a31eed2d497f28c92))
* **tracker:** close GCV-0001, publication scan narrowed and CI unblocked ([b40a3a1](https://github.com/rknightion/grafana-cloud-vending-machine/commit/b40a3a1c7226be07a2a67003629736a45b66ce2b))


### Build System

* migrate task surface to just ([1a07424](https://github.com/rknightion/grafana-cloud-vending-machine/commit/1a07424ad5d36c715f28b8c7610bf96dae240b9e))
* pin the provider to the main build carrying full resource parity ([66e415a](https://github.com/rknightion/grafana-cloud-vending-machine/commit/66e415a07b5282869fef5a1f8d0acc3e6a331748))
* upgrade to Go 1.27 ([092f1ca](https://github.com/rknightion/grafana-cloud-vending-machine/commit/092f1ca2fe0c4f522fba4f06a65d473e023a402b))
