/*
  backgrid-paginator
  http://github.com/wyuenho/backgrid

  Copyright (c) 2013 Jimmy Yuen Ho Wong and contributors
  Licensed under the MIT @license.
*/

(function ($, _, Backbone, Backgrid) {

  "use strict";

  /**
     Paginator is a Backgrid extension that renders a series of configurable
     pagination handles. This extension is best used for splitting a large data
     set across multiple pages. If the number of pages is larger then a
     threshold, which is set to 10 by default, the page handles are rendered
     within a sliding window, plus the fast forward, fast backward, previous and
     next page handles. The fast forward, fast backward, previous and next page
     handles can be turned off.

     @class Backgrid.Extension.Paginator
  */
  Backgrid.Extension.Paginator = Backbone.View.extend({

    /* Rather than use the backgrid-paginator class, which squished
    really large page numbers together, I found the bootstrap
    pagination class to work better. So I took away this class and
    gave the ul a pagination class */

    /** @property */
    windowSize: 10,

    /**
       @property {Object} fastForwardHandleLabels You can disable specific
       handles by setting its value to `null`.
    */
    fastForwardHandleLabels: {
      first: "《",
      prev: "〈",
      next: "〉",
      last: "》"
    },

    /** @property */
    template: _.template('<ul class="pagination"><% _.each(handles, function (handle) { %><li <% if (handle.className) { %>class="<%= handle.className %>"<% } %>><a href="#" <% if (handle.title) {%> title="<%= handle.title %>"<% } %>><%= handle.label %></a></li><% }); %></ul>'),

    /** @property */
    events: {
      "click a": "changePage"
    },

    /**
       Initializer.

       @param {Object} options
       @param {Backbone.Collection} options.collection
       @param {boolean} [options.fastForwardHandleLabels] Whether to render fast forward buttons.
    */
    initialize: function (options) {
      Backgrid.requireOptions(options, ["collection"]);

      /* I took out the fullCollection stuff here, because it seemed
      to cause a bug with the newest version of the backbone-pageable
      collection, where it wouldn't trigger the handle change when you
      clicked on a different page. I couldn't use the old version
      (that's an asset in the backgrid vendor lib) of
      backbone-pageable, since that has a bug that occurs when you
      combine the paginator and filter (it throws a RangeError instead
      of switching to page 1 whenever you search). I don't really know
      why disabling the fullCollection handler option works, but no
      problems so far. */
      var collection = this.collection;
      this.listenTo(collection, "add", this.render);
      this.listenTo(collection, "remove", this.render);
      this.listenTo(collection, "reset", this.render);
    },

    /**
       jQuery event handler for the page handlers. Goes to the right page upon
       clicking.

       @param {Event} e
     */
    changePage: function (e) {
      e.preventDefault();

      var $li = $(e.target).parent();
      if (!$li.hasClass("active") && !$li.hasClass("disabled")) {

        var label = $(e.target).text();
        var ffLabels = this.fastForwardHandleLabels;

        var collection = this.collection;

        if (ffLabels) {
          switch (label) {
          case ffLabels.first:
            collection.getFirstPage();
            return;
          case ffLabels.prev:
            collection.getPreviousPage();
            return;
          case ffLabels.next:
            collection.getNextPage();
            return;
          case ffLabels.last:
            collection.getLastPage();
            return;
          }
        }

        var state = collection.state;
        var pageIndex = +label;
        collection.getPage(state.firstPage === 0 ? pageIndex - 1 : pageIndex);
      }
    },

    /**
       Internal method to create a list of page handle objects for the template
       to render them.

       @return {Array.<Object>} an array of page handle objects hashes
     */
    makeHandles: function () {

      var handles = [];
      var collection = this.collection;
      var state = collection.state;

      // convert all indices to 0-based here
      var firstPage = state.firstPage;
      var lastPage = +state.lastPage;
      lastPage = Math.max(0, firstPage ? lastPage - 1 : lastPage);
      var currentPage = Math.max(state.currentPage, state.firstPage);
      currentPage = firstPage ? currentPage - 1 : currentPage;
      var windowStart = Math.floor(currentPage / this.windowSize) * this.windowSize;
      var windowEnd = Math.min(lastPage + 1, windowStart + this.windowSize);

      if (collection.mode !== "infinite") {
        for (var i = windowStart; i < windowEnd; i++) {
          handles.push({
            label: i + 1,
            title: "No. " + (i + 1),
            className: currentPage === i ? "active" : undefined
          });
        }
      }

      var ffLabels = this.fastForwardHandleLabels;
      if (ffLabels) {

        if (ffLabels.prev) {
          handles.unshift({
            label: ffLabels.prev,
            className: collection.hasPrevious() ? void 0 : "disabled"
          });
        }

        if (ffLabels.first) {
          handles.unshift({
            label: ffLabels.first,
            className: collection.hasPrevious() ? void 0 : "disabled"
          });
        }

        if (ffLabels.next) {
          handles.push({
            label: ffLabels.next,
            className: collection.hasNext() ? void 0 : "disabled"
          });
        }

        if (ffLabels.last) {
          handles.push({
            label: ffLabels.last,
            className: collection.hasNext() ? void 0 : "disabled"
          });
        }
      }

      return handles;
    },

    /**
       Render the paginator handles inside an unordered list.
    */
    render: function () {
      this.$el.empty();

      this.$el.append(this.template({
        handles: this.makeHandles()
      }));

      this.delegateEvents();

      return this;
    }

  });

}(jQuery, _, Backbone, Backgrid));
