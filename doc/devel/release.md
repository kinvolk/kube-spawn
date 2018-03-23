# kube-spawn release guide

## Release cycle

This section describes the typical release cycle of kube-spawn:

1. A GitHub [milestone][milestones] sets the target date for a future kube-spawn release.
2. Issues grouped into the next release milestone are worked on in order of priority.
3. Changes are submitted for review in the form of a GitHub Pull Request (PR). Each PR undergoes review and must pass continuous integration (CI) tests before being accepted and merged into the main line of kube-spawn source code.
4. The day before each release is a short code freeze during which no new code or dependencies may be merged. Instead, this period focuses on polishing the release, with tasks concerning:
  * Documentation
  * Usability tests
  * Issues triaging
  * Roadmap planning and scheduling the next release milestone
  * Organizational and backlog review
  * Build, distribution, and install testing by release manager

## Release process

This section shows how to perform a release of kube-spawn.
Only parts of the procedure are automated; this is somewhat intentional (manual steps for sanity checking) but it can probably be further scripted, help is appreciated.
The following example assumes we're going from version 0.1.1 (`v0.1.1`) to 0.2.0 (`v0.2.0`).

Let's get started:

- Start at the relevant milestone on GitHub (e.g. https://github.com/kinvolk/kube-spawn/milestones/v0.2.0): ensure all referenced issues are closed (or moved elsewhere, if they're not done).
- Make sure your git status is clean: `git status`
- Create a tag locally: `git tag v0.2.0 -m "kube-spawn v0.2.0"`
- Build the release:
  - `git clean -ffdx && make` should work
  - check that the version is correct: `./kube-spawn --version`
  - smoke test the release
  - Integration tests on CI should be green
  - Run the [CNCF conformance tests][conformance-tests] and keep the results
- Prepare the release notes. See [the previous release notes][release-notes] for example.
  Try to capture most of the salient changes since the last release, but don't go into unnecessary detail (better to link/reference the documentation wherever possible).

Push the tag to GitHub:

- Push the tag to GitHub: `git push --tags`

Now we switch to the GitHub web UI to conduct the release:

- Start a [new release][gh-new-release] on Github
- Tag "v0.2.0", release title "v0.2.0"
- Copy-paste the release notes you prepared earlier
- Attach the release.
  This is a simple tarball:

```
export KSVER="0.2.0"
export NAME="kube-spawn-v$KSVER"
mkdir $NAME
cp kube-spawn $NAME/
sudo chown -R root:root $NAME/
tar czvf $NAME.tar.gz --numeric-owner $NAME/
```

- Ensure the milestone on GitHub is closed (e.g. https://github.com/kinvolk/kube-spawn/milestones/v0.2.0)

- Publish the release!

- Submit the [CNCF conformance tests results][conformance-tests]

- Clean your git tree: `sudo git clean -ffdx`.

[conformance-tests]: https://github.com/cncf/k8s-conformance/blob/master/instructions.md
[release-notes]: https://github.com/kinvolk/kube-spawn/releases
[gh-new-release]: https://github.com/kinvolk/kube-spawn/releases/new
[milestones]: https://github.com/kinvolk/kube-spawn/milestones
