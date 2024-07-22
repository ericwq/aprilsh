# commit, branch, issue summary rules

## branch merge
```sh
fix($module): one line brief description
# complex bug fix, which usually involves several modules
#
# examples:
# fix(java): fix config_overrides for tests
# fix(lualine): use the new ministarter file type to disable in mini.starter
```
```sh
chore($module): one line brief description
# release a version or change project scope configuration.
#
# examples:
# chore(main): release 12.26.2
# chore(update): update repository
```
```sh
feat($module): one line brief description
# medium or complex feature
#
# examples:
# feat(java): allow overriding test config
# feat(icons): provide language specific icons in extras
# feat(fzf-lua): preview keymaps for git
```
```sh
refactor($module): one line brief description# complex refactor
# complex refactor, usually involves several modules
#
# examples:
# refactor(trouble): move options in keymaps for lsp and symbols to opts
```
```sh
doc: one line brief description
# document work
#
# examples
# doc: Add recipe to immediately display formatter error output
```
## issue
```sh
bug($module): one line description
# some small, normal bug
#
# examples:
# bug: Pairs inside mini.pairs closeopen don't close inner pair
# bug: relative line numbers doesnt work
# bug: noice causes UI bug when using neovim via ssh
```
```sh
feat($module): one line description
# functional requirement or non-functional requirement
#
# examples:
# feat(extra): add elixirls code actions
# feat(ui): add codewindow (minimap)
# feat(java): allow opting out default keymaps
```

