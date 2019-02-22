luci.project(
    name = 'project',
    milo = 'luci-milo.appspot.com',
)

luci.list_view(name = 'Some view')
luci.console_view(name = 'Some view', repo = 'https://some.repo')

# Expect errors like:
#
# Traceback (most recent call last):
#   //testdata/errors/view_name_clashes.star:7: in <toplevel>
#   ...
# Error: luci.milo_view("Some view") is redeclared, previous declaration:
# Traceback (most recent call last):
#   //testdata/errors/view_name_clashes.star:6: in <toplevel>
#   ...
