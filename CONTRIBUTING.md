# Notice to external contributors


## General info

Hello! In order for us (YANDEX LLC) to accept patches and other contributions from you, you will have to adopt our Yandex Contributor License Agreement (the **CLA**). The current version of the CLA can be found here:
1) https://yandex.ru/legal/cla/?lang=en (in English) and
2) https://yandex.ru/legal/cla/?lang=ru (in Russian).

By adopting the CLA, you state the following:

* You obviously wish and are willingly licensing your contributions to us for our open source projects under the terms of the CLA,
* You have read the terms and conditions of the CLA and agree with them in full,
* You are legally able to provide and license your contributions as stated,
* We may use your contributions for our open source projects and for any other our project too,
* We rely on your assurances concerning the rights of third parties in relation to your contributions.

If you agree with these principles, please read and adopt our CLA. By providing us your contributions, you hereby declare that you have already read and adopt our CLA, and we may freely merge your contributions with our corresponding open source project and use it in further in accordance with terms and conditions of the CLA.

## Provide contributions<a id='1.0'></a>

If you have already adopted terms and conditions of the CLA, you are able to provide your contributions. When you submit your pull request, please add the following information into it:

```
I here by agree to the terms of the CLA available at: [link].
```

Replace the bracketed text as follows:
* [link] is the link to the current version of the CLA: https://yandex.ru/legal/cla/?lang=en (in English) or https://yandex.ru/legal/cla/?lang=ru (in Russian).

It is enough to provide us such notification once.

## Style Guide

We follow [Google Style Guide](https://google.github.io/styleguide/go) but with strict additional commentaries-related rules.

Please pay special attention on:
* [Naming](https://google.github.io/styleguide/go/guide#naming) - it should be self-describing and obvious.
* [Commentaries](https://google.github.io/styleguide/go/decisions#commentary):
  * every public method **required** to have comment in [godoc compatible format](https://tip.golang.org/doc/comment).
  * every package should have `doc.go` file with commentary about what this package means to do.
* All brand-specific names (except "Yandex" for sure) blacklisted.

## Pull Request Process

1. Run `make verify` to apply formatters and start linters.
2. Update the README.md with details of new feature if appropriate.
3. Once all outstanding comments and checklist items have been addressed, your contribution will be merged! Merged PRs will be included in the next release.

## Checklists for contributions

- [ ] Your code covered by tests
- [ ] Your code fulfills Style Guide
- [ ] You do not have any brand-specific names (except Yandex) in your code
- [ ] You successfully have passed `make verify` command
- [ ] README.md has been updated after any changes in "features" or "how to use" description
- [ ] Done steps from [Provide contributions](#1.0) section