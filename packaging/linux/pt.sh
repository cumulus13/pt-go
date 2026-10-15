# Added by the pt-go package: put the pt install directory on the system PATH.
# Sourced by /etc/profile for login shells (Debian/Ubuntu: bash/dash, Alpine: ash).
case ":${PATH}:" in
    *:/TOOLS/pt:*) ;;
    *) PATH="${PATH:+${PATH}:}/TOOLS/pt"; export PATH ;;
esac
