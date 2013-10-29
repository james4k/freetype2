#undef __FTERRORS_H__
#define FT_ERRORDEF(e, v, s) case v: return s;
#define FT_ERROR_START_LIST \
	static const char *ft_error_string(FT_Error code) { \
		switch (code) {
#define FT_ERROR_END_LIST \
	default: return "unknown error"; } }
#include FT_ERRORS_H

