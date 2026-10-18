# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.0] - 2026-10-18

### Changed

- **deps:** Bump vukem to v0.28.0 and namespace the abacatepay webhook dedupe keys ([f418cf7](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/f418cf70ab776eefab8e81f1df585ca8eb65c59f))

## [0.7.0] - 2026-10-18

### Changed

- **deps:** Bump vukem to v0.27.0 with an Inertia Error page ([931b32a](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/931b32a58aef01f3b86750936fff5cfbad5410a9))
- **security:** Use ratelimit.ByIP now that it groups IPv6 by /64 ([4b8ec25](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/4b8ec25c4f0f87da447fbd395075f8c5abdeb3ee))
- **payments:** Count pending refunds with MIN and database.ScanTime ([dc09052](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/dc09052193be91034c29fbf994bc439afeb32759))

### Fixed

- **auth:** Require the account's login to verify its email ([8c537e5](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/8c537e59cccd767d7aaccad9b71af95a9f4c9b85))

## [0.6.0] - 2026-10-17

### Added

- **raffle:** Let visitors buy free numbers from the public page ([3e249f4](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/3e249f4a754a79139c47ad086065876add011d09))
- **raffle:** Show the organizer and opt-in contacts on the public page ([cba34e7](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/cba34e7d37b77b8af5192559f0c1b4fafd3a89ac))
- **payments:** Admin list of refunds due with mark-refunded action ([8dc6328](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/8dc6328a75a2a497d801432e859f1662a9e2cb79))

### Changed

- **numbers:** Load purchase raffles in one query ([9b3a873](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/9b3a873ecc84cb2503a82c8fbc5e721cd0175963))

### Fixed

- **ui:** Collapse the navbar links into a menu on phones ([1c0e53b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/1c0e53bb44ab6fcd79c3ffe4fc4bdad8095739fb))
- **security:** Record LGPD consent on public reservations ([1fbab1f](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/1fbab1fe6bb85ffbebf3e21cf791a5310faecbcd))
- **raffle:** Lock the raffle row before editing it ([06f68b6](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/06f68b6ba329f3bdcd4c72eb480b0388b6c577a8))
- **numbers:** Reserve against the locked raffle row, not the caller's copy ([cbbed30](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/cbbed3013fd765613febcb1db88a8d2cf27636c2))
- **security:** Cap live public links per raffle and fold email aliases ([edbd0c3](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/edbd0c3badc48e0a4391d026c1e9ade2f6d67bc7))
- **security:** Erase the sealed link token when the link ends ([8fe2182](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/8fe21823d5b5a9876eca034421df19d47aaacf90))
- **security:** Reveal the organizer email only through a limited endpoint ([e758df7](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/e758df7a37f51d4363901334097b19cd685090fd))

## [0.5.0] - 2026-10-17

### Added

- **ui:** Match the navbar container to the vukem docs site ([437155c](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/437155caa1e877baedd510ea02953824ed264691))
- **ui:** Theme toggle with sun, moon and system icons like the docs ([f3d74e2](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/f3d74e2a42b72eaa197ed079b4a6a0383ea3447f))
- **raffle:** Cancel live payment links when the owner edits a raffle ([862c721](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/862c72195f3940efc4a37b6c1fe0aa46d02c3b4a))
- **numbers:** Keep live payment links visible to the owner for 10 minutes ([5d4650b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/5d4650b1e9a7593e700daa7b41300e6b9eb09285))
- **payments:** Show the PIX QR inline and copy the code with clipboard.js ([1e7c130](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/1e7c1303acdc5503569b8118671f1db40ec41cfd))

### Changed

- **deps:** Bump vukem to v0.26.0 ([191bf94](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/191bf94290ec2291168a57fb82cf9efc354ebc71))

## [0.4.0] - 2026-10-17

### Changed

- **deps:** Bump vukem to v0.25.0 ([7794ae8](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/7794ae8bdb518cb8b4350294aef81b7311f942ae))

## [0.3.0] - 2026-10-17

### Changed

- **deps:** Bump vukem to v0.24.0 ([86c7df5](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/86c7df59e89722021ff642a7b065d6d623f0b4d2))

### Security

- **security:** Make sign-up enumeration-safe and close A02/A05/A09 gaps ([271cd2b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/271cd2b9d8a7219d5a0460b2ac910d1b8b332807))

## [0.2.0] - 2026-10-17

### Added

- **raffle:** Add raffle management and public page ([b9c5107](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/b9c5107d65f43b0207972ea15b4443476ae498da))
- **db:** Add reservations, payments, ledger and draws schema ([a0bd558](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/a0bd5583272d2b5ed4ab9e1c17e5edd539dd5d68))
- **numbers:** Reserve numbers with hashed payment links ([c44edf6](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/c44edf652954a8b5a28c2a2fef50798d88e86e5a))
- **api:** Add raffle audit export as JSON and CSV ([2e30078](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/2e300784b245259698f17e4fb73128a6b747d197))
- **api:** Add API keys and REST API v1 with OpenAPI and Scalar docs ([6febf33](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/6febf3321c0923c3d9a9bf8ae6cdb544ceca1b8c))
- **api:** Generate TypeScript types and zod schemas from the OpenAPI document ([106a572](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/106a5729623a2717ea04646976961e2f9b5c4651))
- **ui:** Add API keys settings page and raffle export links ([066921b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/066921b6d49dd8b1c35253216425f14b8ccf757b))
- **draw:** Add auditable v1 draw algorithm ([4b161f8](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/4b161f88668652f295a243e95f07eee0f71f61d9))
- **draw:** Run draws at the deadline and when sold out ([35dbbef](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/35dbbef5dd32e3332bb0e3a5923306871805f94b))
- **numbers:** Generate and cancel payment links from the manage page ([c4e0901](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/c4e09017c2d1033a6ad8154c1656d12782152ac0))
- **ui:** Payment-link panel on the manage page and auditable draw result ([3314728](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/3314728dd8c513161c418dcc553e4efad7fe23c2))
- **payments:** Add AbacatePay v2 PIX client, webhook parser and fake gateway ([88eb4f5](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/88eb4f59edac3d5c38059b76f817b62c2d2f2b2b))
- **payments:** Buyer checkout, PIX confirm and verified AbacatePay webhook with ledger and emails ([5fa43e3](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/5fa43e3c094e89cc9ef3d4427bf58090c25d2d47))
- **ledger:** Owner balance on dashboard, admin payouts and make:admin command ([78556f4](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/78556f4f6f70be74dcd52f5975ece3b495a15b89))
- **ui:** Buyer checkout with PIX modal and polling, owner balance and admin payouts pages ([b99df81](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/b99df815da76bc5323213ca0bfe11946671f2d76))

### Changed

- **deps:** Bump vukem to v0.22.0 ([7137e30](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/7137e306fcdce38465a297339393f62f006b2660))
- **deps:** Bump vukem to v0.23.0 ([8dd7818](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/8dd78181795ffd7bb2ea135b9d3589eb2157176a))
- **numbers:** Build the payment-link limiter from app.RateLimiters ([48f4518](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/48f4518654dc64346b6bd917e6b702fe6d506b4e))

### Fixed

- **security:** Build API and key-creation rate limiters from app.RateLimiters ([8f93f1d](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/8f93f1d90b11ed05bb41b13f259c07b56fd1d65e))
- **i18n:** Stop repeating the currency symbol in the raffle price ([74e112b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/74e112bf70a15bd7bf7fe1e225c9ad1c284fb920))
- **raffle:** Treat closed raffles as finished ([7165faf](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/7165faf95cc5adcc88ca93f51c902c9544a78444))
- **payments:** Lock raffle on payment, refuse closed raffles, cap PIX at ends_at ([423ff07](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/423ff07a59c76ee71c019a1138d1114929f5e947))

## [0.1.0] - 2026-10-17

### Added

- **repo:** Scaffold vukem inertia vue app on postgres ([824eb38](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/824eb3806d4fcf33c2607c231ddd7fc276b80b79))
- **auth:** Install vukem auth with google login ([cb0a30c](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/cb0a30c7d7de4edc5079971a15bb853601ab53e0))
- **auth:** Require a verified email for the dashboard ([21d18fb](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/21d18fb06be71ce0a310afdba17585f2065fb63a))
- **db:** Add is_admin flag to users ([825a266](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/825a2660a51d918dc90ab0b2e4ca50afcb20f12e))
- **i18n:** Translate auth pages, flashes and errors ([31173fe](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/31173fe385975c420ad04147d92414fb756da024))
- **i18n:** Localize validation messages in pt-BR and en ([e1e968b](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/e1e968bad853979aa57630e15c2871ce9dbaca55))
- **mail:** Send auth emails in the request locale ([fa00f06](https://github.com/AlexGalhardo/vukem-rifa-vuejs/commit/fa00f066effd09301d15f85c9f0c71ab6c0590da))

[0.8.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AlexGalhardo/vukem-rifa-vuejs/releases/tag/v0.1.0
