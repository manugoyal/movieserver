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
 * The application called on startup.
 * exports: App
 */

define(['jquery', 'underscore', 'views/movie_table'],
       function ($, _, MovieTableView) {
         var App = {
           offline: false,
           movieTable: null,

           poller: function() {
             $.ajax({
               url: '/',
               success: _.bind(
                 function() {
                   if (this.offline) {
                     this.offline = false;
                     $('#no-connection-alert').hide();
                     if (this.movieTable) {
                       this.movieTable.destroy();
                       this.initialize();
                     }
                   }
                 }, this),
               error: _.bind(
                 function() {
                   $('#no-connection-alert').show();
                   this.offline = true;
                   _.bind(this.poller, this)();
                 }, this)
             });
           },

           initialize: function () {
             $.getJSON('tableKeys/', _.bind(function(data) {
               if (data.length === 0) {
                 $('#no-keys-alert').show();
                 return;
               } else {
                 $('#no-keys-alert').hide();
                 this.movieTable = new MovieTableView({ el: $('.container'), tableKeys: data});
               }
             }, this));
           }
         };
         return App;
       });