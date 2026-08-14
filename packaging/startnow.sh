# StartNow: expose ~/.startnow/bin to login shells.
if [ -d "$HOME/.startnow/bin" ]; then
    case ":${PATH}:" in
        *":$HOME/.startnow/bin:"*) ;;
        *) export PATH="$HOME/.startnow/bin:$PATH" ;;
    esac
fi
