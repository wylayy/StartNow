# StartNow: expose ~/.startnow/bin to fish.
if test -d "$HOME/.startnow/bin"
    and not contains -- "$HOME/.startnow/bin" $PATH
    set -gx PATH "$HOME/.startnow/bin" $PATH
end
