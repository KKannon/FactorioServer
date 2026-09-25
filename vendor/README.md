# Vendored backend dependency

`lib-stupidmailjavascript-0.1.0.tgz` is an `npm pack` artifact built from
`Stupid-DLL/lib-StupidMailJavaScript` commit
`1e0053d381d0a03657e0df9dc4fe6bdf7b844345` (MIT).

The dependency is vendored because the repository is private and the Umbrel
build host intentionally has no GitHub credential. To update it, clone the
library with an authorized account, run `npm ci`, `npm test`, `npm run build`,
and `npm pack --pack-destination <this directory>`. Then update the filename,
integrity entry, and version in `mailer/package.json` and its lockfile.
