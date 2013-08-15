/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
 */

/*
 * Defines an modification of the Backgrid UriCell type, which
 * prepends 'movie/' to the href, so that the webserver can handle it
 * properly.
 * exports: MovieUri
 */

define(['jquery', 'underscore', 'backgrid'], function($, _, Backgrid) {
  var MovieUri = Backgrid.UriCell.extend({
    render: function () {
      this.$el.empty();
      var formattedValue = this.formatter.fromRaw(this.model.get(this.column.get("name")));
      var hrefFormatted = 'movie/' + formattedValue;
      this.$el.append($("<a>", {
        tabIndex: -1,
        href: hrefFormatted,
        title: formattedValue,
        target: "_blank"
      }).text(formattedValue));
      this.delegateEvents();
      return this;
    }
  });

  return MovieUri;
});