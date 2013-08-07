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
 * The view of a movies row in a movies table
 * exports: MovieRowView
 */

/*
 * The view of a movies table, which consists of movie rows, whose
 * model is a table collection
 * exports: MovieTableView
 */

define(['jquery', 'underscore', 'backbone', 'collections/movie_table', 'views/movie_row'],
       function($, _, Backbone, MovieTableCollection, MovieRowView) {
         var MovieTableView = Backbone.View.extend({

           // Maps movie names to their movie row views
           movies: {},

           initialize: function() {
             this.collection = new MovieTableCollection();
             this.listenTo(this.collection, 'add', this._add_elm);
             this.listenTo(this.collection, 'remove', this._remove_elm);
             this.refresh();
           },

           refresh: function() {
             this.collection.fetch();
           },

           _add_elm: function(elm) {
             var movieRowView = new MovieRowView({ model: elm });
             movieRowView.render();
             this.movies[elm.get("Name")] = movieRowView;
             this.$el.append(movieRowView.el);
           },

           _remove_elm: function(elm) {
             var movieRowView = this.movies[elm.get("Name")];
             movieRowView.remove();
             delete this.movies[elm.get("Name")];
           }

         });

         return MovieTableView;
       });