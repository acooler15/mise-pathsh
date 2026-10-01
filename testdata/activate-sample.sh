export __MISE_ORIG_PATH='C:\Users\tester\bin;C:\Program Files\Git\mingw64\bin;C:\Program Files\Git\usr\local\bin;C:\Program Files\Git\usr\bin;C:\Program Files\Git\usr\bin;C:\Program Files\Git\mingw64\bin;C:\Program Files\Git\usr\bin;C:\Users\tester\bin;C:\WINDOWS\system32;C:\WINDOWS;C:\WINDOWS\System32\Wbem;C:\WINDOWS\System32\WindowsPowerShell\v1.0;C:\WINDOWS\System32\OpenSSH;C:\Program Files\dotnet;C:\Program Files (x86)\NVIDIA Corporation\PhysX\Common;C:\Program Files (x86)\NetSarang\Xshell 8;C:\Program Files (x86)\NetSarang\Xftp 8;C:\Program Files (x86)\Windows Kits\10\Windows Performance Toolkit;C:\Program Files\Common Files\Autodesk Shared;C:\Program Files\Microsoft SQL Server\150\Tools\Binn;C:\Program Files\NVIDIA Corporation\NVIDIA app\NvDLISR;C:\Program Files\usbipd-win;C:\Program Files (x86)\Windows Kits\8.1\Windows Performance Toolkit;C:\Program Files\starship\bin;C:\Program Files\PowerShell\7;C:\Program Files\Git\cmd;C:\Program Files\WSL;C:\Program Files\GitHub CLI;C:\Users\tester\scoop\shims;C:\Users\tester\AppData\Local\Programs\Trae CN\bin;E:\develop\mise\shims;C:\Program Files\PowerShell\7;C:\Program Files\NVIDIA Corporation\NVIDIA app\NvDLI;C:\Program Files\Common Files\FAST\CAD;C:\Users\tester\AppData\Local\Microsoft\WindowsApps;C:\Users\tester\AppData\Local\Programs\Microsoft VS Code\bin;C:\Users\tester\AppData\Local\Programs\CodeBuddy CN\bin;C:\Users\tester\AppData\Local\Microsoft\WinGet\Links;C:\Users\tester\AppData\Local\Microsoft\WinGet\Packages\ISC.Bind_Microsoft.Winget.Source_8wekyb3d8bbwe;E:\develop\android\android_sdk_home\platform-tools;C:\Users\tester\AppData\Local\PowerToys\DSCModules;C:\Users\tester\AppData\Local\Programs\Qoder CN IDE\bin;C:\Users\tester\AppData\Local\Programs\ZCode\resources\tools\ripgrep;C:\Users\tester\AppData\Local\Programs\ZCode\resources\tools\ugrep;C:\Program Files\Git\usr\bin\vendor_perl;C:\Program Files\Git\usr\bin\core_perl'
export PATH='E:\develop\mise\shims;C:\Users\tester\bin;C:\Program Files\Git\mingw64\bin;C:\Program Files\Git\usr\local\bin;C:\Program Files\Git\usr\bin;C:\Program Files\Git\usr\bin;C:\Program Files\Git\mingw64\bin;C:\Program Files\Git\usr\bin;C:\Users\tester\bin;C:\WINDOWS\system32;C:\WINDOWS;C:\WINDOWS\System32\Wbem;C:\WINDOWS\System32\WindowsPowerShell\v1.0;C:\WINDOWS\System32\OpenSSH;C:\Program Files\dotnet;C:\Program Files (x86)\NVIDIA Corporation\PhysX\Common;C:\Program Files (x86)\NetSarang\Xshell 8;C:\Program Files (x86)\NetSarang\Xftp 8;C:\Program Files (x86)\Windows Kits\10\Windows Performance Toolkit;C:\Program Files\Common Files\Autodesk Shared;C:\Program Files\Microsoft SQL Server\150\Tools\Binn;C:\Program Files\NVIDIA Corporation\NVIDIA app\NvDLISR;C:\Program Files\usbipd-win;C:\Program Files (x86)\Windows Kits\8.1\Windows Performance Toolkit;C:\Program Files\starship\bin;C:\Program Files\PowerShell\7;C:\Program Files\Git\cmd;C:\Program Files\WSL;C:\Program Files\GitHub CLI;C:\Users\tester\scoop\shims;C:\Users\tester\AppData\Local\Programs\Trae CN\bin;C:\Program Files\PowerShell\7;C:\Program Files\NVIDIA Corporation\NVIDIA app\NvDLI;C:\Program Files\Common Files\FAST\CAD;C:\Users\tester\AppData\Local\Microsoft\WindowsApps;C:\Users\tester\AppData\Local\Programs\Microsoft VS Code\bin;C:\Users\tester\AppData\Local\Programs\CodeBuddy CN\bin;C:\Users\tester\AppData\Local\Microsoft\WinGet\Links;C:\Users\tester\AppData\Local\Microsoft\WinGet\Packages\ISC.Bind_Microsoft.Winget.Source_8wekyb3d8bbwe;E:\develop\android\android_sdk_home\platform-tools;C:\Users\tester\AppData\Local\PowerToys\DSCModules;C:\Users\tester\AppData\Local\Programs\Qoder CN IDE\bin;C:\Users\tester\AppData\Local\Programs\ZCode\resources\tools\ripgrep;C:\Users\tester\AppData\Local\Programs\ZCode\resources\tools\ugrep;C:\Program Files\Git\usr\bin\vendor_perl;C:\Program Files\Git\usr\bin\core_perl'
export PATH="C:\\Users\\tester\\scoop\\apps\\mise\\current\\bin:$PATH"
# shellcheck shell=bash
export __MISE_EXE='C:\Users\tester\scoop\apps\mise\current\bin\mise.exe'
__MISE_FLAGS=()
__MISE_HOOK_ENABLED=1

export MISE_SHELL=bash

# On first activation, save the original PATH
# On re-activation, we keep the saved original
if [ -z "${__MISE_ORIG_PATH:-}" ]; then
	export __MISE_ORIG_PATH="$PATH"
fi
__MISE_BASH_CHPWD_RAN=0

mise() {
	local command
	command="${1:-}"
	if [ "$#" = 0 ]; then
		command 'C:\Users\tester\scoop\apps\mise\current\bin\mise.exe'
		return
	fi
	shift

	case "$command" in
	deactivate | shell | sh)
		# if argv doesn't contains -h,--help
		if [[ ! " $* " =~ " --help " ]] && [[ ! " $* " =~ " -h " ]]; then
			eval "$(command 'C:\Users\tester\scoop\apps\mise\current\bin\mise.exe' "$command" "$@")"
			return $?
		fi
		;;
	esac
	command 'C:\Users\tester\scoop\apps\mise\current\bin\mise.exe' "$command" "$@"
}

_mise_hook() {
	local previous_exit_status=$?
	eval "$(mise hook-env ${__MISE_FLAGS[@]+"${__MISE_FLAGS[@]}"} --shell-pid $$ -s bash "$@")"
	return $previous_exit_status
}

if [ "$__MISE_HOOK_ENABLED" = "1" ]; then
	_mise_hook_prompt_command() {
		local previous_exit_status=$?
		if [[ ${__MISE_BASH_CHPWD_RAN:-0} == "1" ]]; then
			__MISE_BASH_CHPWD_RAN=0
			unset __MISE_BASH_SKIP_FIRST_PROMPT
			return $previous_exit_status
		fi
		if [[ ${__MISE_BASH_SKIP_FIRST_PROMPT:-0} == "1" ]]; then
			unset __MISE_BASH_SKIP_FIRST_PROMPT
			return $previous_exit_status
		fi
		eval "$(mise hook-env ${__MISE_FLAGS[@]+"${__MISE_FLAGS[@]}"} --shell-pid $$ -s bash --reason precmd)"
		return $previous_exit_status
	}

	_mise_hook_chpwd() {
		local previous_exit_status=$?
		__MISE_BASH_CHPWD_RAN=1
		eval "$(mise hook-env ${__MISE_FLAGS[@]+"${__MISE_FLAGS[@]}"} --shell-pid $$ -s bash --reason chpwd)"
		return $previous_exit_status
	}

	_mise_add_prompt_command() {
		if [[ "$(declare -p PROMPT_COMMAND 2>/dev/null)" == "declare -a"* ]]; then
			if [[ " ${PROMPT_COMMAND[*]} " != *" _mise_hook_prompt_command "* ]]; then
				PROMPT_COMMAND=("_mise_hook_prompt_command" "${PROMPT_COMMAND[@]}")
			fi
		elif [[ ";${PROMPT_COMMAND:-};" != *";_mise_hook_prompt_command;"* ]]; then
			local _mise_prompt_command_value="${PROMPT_COMMAND-}"
			printf -v PROMPT_COMMAND '%s' "_mise_hook_prompt_command${_mise_prompt_command_value:+;$_mise_prompt_command_value}"
		fi
	}

	_mise_add_prompt_command
	# shellcheck shell=bash
export -a chpwd_functions
function __zsh_like_cd()
{
  \typeset __zsh_like_cd_hook
  if
    builtin "$@"
  then
    for __zsh_like_cd_hook in chpwd ${chpwd_functions[@]+"${chpwd_functions[@]}"}
    do
      if \typeset -f "$__zsh_like_cd_hook" >/dev/null 2>&1
      then "$__zsh_like_cd_hook" || break # finish on first failed hook
      fi
    done
    true
  else
    return $?
  fi
}

	# shellcheck shell=bash
[[ -n "${ZSH_VERSION:-}" ]] ||
{
  function cd()    { __zsh_like_cd cd    "$@" ; }
  function popd()  { __zsh_like_cd popd  "$@" ; }
  function pushd() { __zsh_like_cd pushd "$@" ; }
}

	chpwd_functions+=(_mise_hook_chpwd)
fi

# `--no-hook-env` means mise does not apply the environment at activation -- zsh, fish and pwsh
# all keep their equivalent call inside this guard, and bash did too until the script moved out
# of bash.rs into this file (#8920), which left the call outside it. The skip flag exists because
# the hook runs here, so the two belong under one condition; it is set first so the `$?` that
# `_mise_hook` saves is not read straight off the guard's condition, where it is the test's
# status rather than a command's (SC2319).
if [ "$__MISE_HOOK_ENABLED" = "1" ]; then
	__MISE_BASH_SKIP_FIRST_PROMPT=1
	_mise_hook --force
fi

# shellcheck shell=bash
if [ -z "${_mise_cmd_not_found:-}" ]; then
	_mise_cmd_not_found=1
	if [ -n "$(declare -f command_not_found_handle)" ]; then
		_mise_cmd_not_found_handle=$(declare -f command_not_found_handle)
		eval "${_mise_cmd_not_found_handle/command_not_found_handle/_command_not_found_handle}"
	fi

	command_not_found_handle() {
		if [[ $1 != "mise" && $1 != "mise-"* ]] && 'C:\Users\tester\scoop\apps\mise\current\bin\mise.exe' hook-not-found -s bash -- "$1"; then
			_mise_hook
			"$@"
		elif [ -n "$(declare -f _command_not_found_handle)" ]; then
			_command_not_found_handle "$@"
		else
			echo "bash: command not found: $1" >&2
			return 127
		fi
	}
fi

