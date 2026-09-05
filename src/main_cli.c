/*
 * main_cli.c - headless Workbook engine driver ("cell").
 *
 * Usage:
 *   cell eval FILE [-o OUT.csv]       evaluate and export the first sheet as CSV
 *   cell csv2gnm IN.csv OUT.gnumeric  import CSV, write a .gnumeric file
 *   cell gnm2csv IN.gnumeric OUT.csv  load .gnumeric, evaluate, export CSV
 *   cell info FILE                    sheet names and dimensions
 */
#include "src/engine.h"

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

static void
usage(void)
{
    fprintf(stderr,
            "usage: cell eval FILE [-o OUT.csv]\n"
            "       cell csv2gnm IN.csv OUT.gnumeric\n"
            "       cell gnm2csv IN.gnumeric OUT.csv\n"
            "       cell info FILE\n");
}

static int
cmd_eval(const char *in, const char *out)
{
    Workbook *wb = WorkbookNew();

    if(wb == NULL)
        return 1;
    if(!WorkbookLoadFile(wb, in)) {
        fprintf(stderr, "cell: cannot load %s\n", in);
        WorkbookFree(wb);
        return 1;
    }
    if(!WorkbookExportCsv(wb, out)) {
        fprintf(stderr, "cell: cannot write %s\n", out);
        WorkbookFree(wb);
        return 1;
    }
    WorkbookFree(wb);
    return 0;
}

static int
cmd_csv2gnm(const char *in, const char *out)
{
    Workbook *wb = WorkbookNew();

    if(wb == NULL)
        return 1;
    if(!WorkbookLoadFile(wb, in)) {
        fprintf(stderr, "cell: cannot load %s\n", in);
        WorkbookFree(wb);
        return 1;
    }
    if(!WorkbookWriteGnumeric(wb, out)) {
        fprintf(stderr, "cell: cannot write %s\n", out);
        WorkbookFree(wb);
        return 1;
    }
    WorkbookFree(wb);
    return 0;
}

static int
cmd_info(const char *in)
{
    Workbook *wb = WorkbookNew();
    int i;

    if(wb == NULL)
        return 1;
    if(!WorkbookLoadFile(wb, in)) {
        fprintf(stderr, "cell: cannot load %s\n", in);
        WorkbookFree(wb);
        return 1;
    }
    printf("%d sheet(s)\n", WorkbookSheetCount(wb));
    for(i = 0; i < WorkbookSheetCount(wb); i++)
        printf("%s: %d rows x %d cols\n", WorkbookSheetName(wb, i),
               WorkbookSheetRows(wb, i), WorkbookSheetColumns(wb, i));
    WorkbookFree(wb);
    return 0;
}

int
main(int argc, char **argv)
{
    if(argc < 3) {
        usage();
        return 2;
    }
    if(strcmp(argv[1], "eval") == 0) {
        const char *out = argc >= 5 && strcmp(argv[3], "-o") == 0
                              ? argv[4]
                              : "-";

        if(argc < 3) {
            usage();
            return 2;
        }
        if(strcmp(out, "-") == 0) {
            char tmp[4096];

            snprintf(tmp, sizeof(tmp), "/tmp/cell-eval-%d.csv", (int)getpid());
            if(cmd_eval(argv[2], tmp) != 0)
                return 1;
            {
                FILE *f = fopen(tmp, "r");
                char buf[8192];
                size_t n;

                if(f == NULL)
                    return 1;
                while((n = fread(buf, 1, sizeof(buf), f)) != 0)
                    fwrite(buf, 1, n, stdout);
                fclose(f);
                remove(tmp);
            }
            return 0;
        }
        return cmd_eval(argv[2], out);
    }
    if(strcmp(argv[1], "info") == 0 && argc == 3)
        return cmd_info(argv[2]);
    if(strcmp(argv[1], "copy") == 0 && argc == 4)
        return cmd_csv2gnm(argv[2], argv[3]);
    if(argc < 4) {
        usage();
        return 2;
    }
    if(strcmp(argv[1], "csv2gnm") == 0)
        return cmd_csv2gnm(argv[2], argv[3]);
    if(strcmp(argv[1], "gnm2csv") == 0)
        return cmd_eval(argv[2], argv[3]);
    if(strcmp(argv[1], "info") == 0)
        return cmd_info(argv[2]);
    usage();
    return 2;
}
